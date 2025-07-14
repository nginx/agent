// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cenkalti/backoff/v4"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/id"

	"google.golang.org/protobuf/types/known/timestamppb"

	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
)

var _ commandService = (*CommandService)(nil)

const (
	createConnectionMaxElapsedTime = 0
)

type (
	CommandService struct {
		commandServiceClient         mpi.CommandServiceClient
		subscribeClient              mpi.CommandService_SubscribeClient
		agentConfig                  *config.Config
		isConnected                  *atomic.Bool
		subscribeChannel             chan *mpi.ManagementPlaneRequest
		configApplyRequestQueue      map[string][]*mpi.ManagementPlaneRequest // key is the instance ID
		resource                     *mpi.Resource
		subscribeClientMutex         sync.Mutex
		configApplyRequestQueueMutex sync.Mutex
		resourceMutex                sync.Mutex
	}
)

func NewCommandService(
	commandServiceClient mpi.CommandServiceClient,
	agentConfig *config.Config,
	subscribeChannel chan *mpi.ManagementPlaneRequest,
) *CommandService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	commandService := &CommandService{
		commandServiceClient:    commandServiceClient,
		agentConfig:             agentConfig,
		isConnected:             isConnected,
		subscribeChannel:        subscribeChannel,
		configApplyRequestQueue: make(map[string][]*mpi.ManagementPlaneRequest),
		resource:                &mpi.Resource{},
	}

	return commandService
}

func (cs *CommandService) IsConnected() bool {
	return cs.isConnected.Load()
}

func (cs *CommandService) UpdateDataPlaneStatus(
	ctx context.Context,
	resource *mpi.Resource,
) error {
	correlationID := logger.CorrelationID(ctx)
	if !cs.isConnected.Load() {
		return errors.New("command service client not connected yet")
	}

	request := &mpi.UpdateDataPlaneStatusRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendDataPlaneStatus := func() (*mpi.UpdateDataPlaneStatusResponse, error) {
		slog.DebugContext(ctx, "Sending data plane status update request", "request", request,
			"parent_correlation_id", correlationID)

		cs.subscribeClientMutex.Lock()
		if cs.commandServiceClient == nil {
			cs.subscribeClientMutex.Unlock()
			return nil, errors.New("command service client is not initialized")
		}
		response, updateError := cs.commandServiceClient.UpdateDataPlaneStatus(ctx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(updateError)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update data plane status", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendDataPlaneStatus,
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Client.Backoff),
	)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "UpdateDataPlaneStatus response", "response", response)

	cs.resourceMutex.Lock()
	defer cs.resourceMutex.Unlock()
	cs.resource = resource

	return err
}

func (cs *CommandService) UpdateDataPlaneHealth(ctx context.Context, instanceHealths []*mpi.InstanceHealth) error {
	if !cs.isConnected.Load() {
		return errors.New("command service client not connected yet")
	}

	correlationID := logger.CorrelationID(ctx)

	request := &mpi.UpdateDataPlaneHealthRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		InstanceHealths: instanceHealths,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	response, err := backoff.RetryWithData(
		cs.dataPlaneHealthCallback(ctx, request),
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Client.Backoff),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateDataPlaneHealth response", "response", response)

	return err
}

func (cs *CommandService) SendDataPlaneResponse(ctx context.Context, response *mpi.DataPlaneResponse) error {
	slog.DebugContext(ctx, "Sending data plane response", "response", response)

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	err := cs.handleConfigApplyResponse(ctx, response)
	if err != nil {
		return err
	}

	return backoff.Retry(
		cs.sendDataPlaneResponseCallback(ctx, response),
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Client.Backoff),
	)
}

func (cs *CommandService) Subscribe(ctx context.Context) {
	commonSettings := &config.BackOff{
		InitialInterval:     cs.agentConfig.Client.Backoff.InitialInterval,
		MaxInterval:         cs.agentConfig.Client.Backoff.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Client.Backoff.RandomizationFactor,
		Multiplier:          cs.agentConfig.Client.Backoff.Multiplier,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			retryError := backoff.Retry(cs.receiveCallback(ctx), backoffHelpers.Context(ctx, commonSettings))
			if retryError != nil {
				slog.WarnContext(ctx, "Failed to receive messages from subscribe stream", "error", retryError)
			}
		}
	}
}

func (cs *CommandService) CreateConnection(
	ctx context.Context,
	resource *mpi.Resource,
) (*mpi.CreateConnectionResponse, error) {
	correlationID := logger.CorrelationID(ctx)
	if len(resource.GetInstances()) <= 1 {
		slog.InfoContext(ctx, "No Data Plane Instance found")
	}

	if cs.isConnected.Load() {
		return nil, errors.New("command service already connected")
	}

	request := &mpi.CreateConnectionRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	commonSettings := &config.BackOff{
		InitialInterval:     cs.agentConfig.Client.Backoff.InitialInterval,
		MaxInterval:         cs.agentConfig.Client.Backoff.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Client.Backoff.RandomizationFactor,
		Multiplier:          cs.agentConfig.Client.Backoff.Multiplier,
	}

	slog.DebugContext(ctx, "Sending create connection request", "request", request)
	response, err := backoff.RetryWithData(
		cs.connectCallback(ctx, request),
		backoffHelpers.Context(ctx, commonSettings),
	)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Connection created", "response", response)
	slog.InfoContext(ctx, "Agent connected")

	cs.isConnected.Store(true)

	cs.resourceMutex.Lock()
	defer cs.resourceMutex.Unlock()
	cs.resource = resource

	return response, nil
}

func (cs *CommandService) UpdateClient(ctx context.Context, client mpi.CommandServiceClient) error {
	cs.subscribeClientMutex.Lock()
	cs.commandServiceClient = client
	cs.subscribeClientMutex.Unlock()

	cs.isConnected.Store(false)

	resp, err := cs.CreateConnection(ctx, cs.resource)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "Successfully sent create connection request", "response", resp)

	return nil
}

// Retry callback for sending a data plane response to the Management Plane.
func (cs *CommandService) sendDataPlaneResponseCallback(
	ctx context.Context,
	response *mpi.DataPlaneResponse,
) func() error {
	return func() error {
		cs.subscribeClientMutex.Lock()
		defer cs.subscribeClientMutex.Unlock()

		if cs.subscribeClient == nil {
			return errors.New("subscribe client is not initialized")
		}

		err := cs.subscribeClient.Send(response)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send data plane response", "error", err)

			return err
		}

		return nil
	}
}

func (cs *CommandService) handleConfigApplyResponse(
	ctx context.Context,
	response *mpi.DataPlaneResponse,
) error {
	cs.configApplyRequestQueueMutex.Lock()
	defer cs.configApplyRequestQueueMutex.Unlock()

	isConfigApplyResponse := false
	var indexOfConfigApplyRequest int

	for index, configApplyRequest := range cs.configApplyRequestQueue[response.GetInstanceId()] {
		if configApplyRequest.GetMessageMeta().GetCorrelationId() == response.GetMessageMeta().GetCorrelationId() {
			indexOfConfigApplyRequest = index
			isConfigApplyResponse = true

			break
		}
	}

	if isConfigApplyResponse {
		err := cs.sendResponseForQueuedConfigApplyRequests(ctx, response, indexOfConfigApplyRequest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cs *CommandService) sendResponseForQueuedConfigApplyRequests(
	ctx context.Context,
	response *mpi.DataPlaneResponse,
	indexOfConfigApplyRequest int,
) error {
	instanceID := response.GetInstanceId()
	for i := range indexOfConfigApplyRequest {
		newResponse := response

		newResponse.GetMessageMeta().MessageId = id.GenerateMessageID()

		request := cs.configApplyRequestQueue[instanceID][i]
		newResponse.GetMessageMeta().CorrelationId = request.GetMessageMeta().GetCorrelationId()

		slog.DebugContext(
			ctx,
			"Sending data plane response for queued config apply request",
			"response", newResponse,
		)

		backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Client.Backoff.MaxElapsedTime)

		err := backoff.Retry(
			cs.sendDataPlaneResponseCallback(ctx, newResponse),
			backoffHelpers.Context(backOffCtx, cs.agentConfig.Client.Backoff),
		)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send data plane response", "error", err)
			backoffCancel()

			return err
		}

		backoffCancel()
	}

	cs.configApplyRequestQueue[instanceID] = cs.configApplyRequestQueue[instanceID][indexOfConfigApplyRequest+1:]
	slog.DebugContext(ctx, "Removed config apply requests from queue", "queue", cs.configApplyRequestQueue[instanceID])

	if len(cs.configApplyRequestQueue[instanceID]) > 0 {
		cs.subscribeChannel <- cs.configApplyRequestQueue[instanceID][len(cs.configApplyRequestQueue[instanceID])-1]
	}

	return nil
}

// Retry callback for sending a data plane health status to the Management Plane.
func (cs *CommandService) dataPlaneHealthCallback(
	ctx context.Context,
	request *mpi.UpdateDataPlaneHealthRequest,
) func() (*mpi.UpdateDataPlaneHealthResponse, error) {
	return func() (*mpi.UpdateDataPlaneHealthResponse, error) {
		slog.DebugContext(ctx, "Sending data plane health update request", "request", request)

		cs.subscribeClientMutex.Lock()
		if cs.commandServiceClient == nil {
			cs.subscribeClientMutex.Unlock()
			return nil, errors.New("command service client is not initialized")
		}

		response, updateError := cs.commandServiceClient.UpdateDataPlaneHealth(ctx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update data plane health", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}
}

// Retry callback for receiving messages from the Management Plane subscription.
// nolint: revive
func (cs *CommandService) receiveCallback(ctx context.Context) func() error {
	return func() error {
		cs.subscribeClientMutex.Lock()

		if cs.subscribeClient == nil {
			if cs.commandServiceClient == nil {
				cs.subscribeClientMutex.Unlock()
				return errors.New("command service client is not initialized")
			}

			var err error
			cs.subscribeClient, err = cs.commandServiceClient.Subscribe(ctx)
			if err != nil {
				subscribeErr := cs.handleSubscribeError(ctx, err, "create subscribe client")
				cs.subscribeClientMutex.Unlock()

				return subscribeErr
			}

			if cs.subscribeClient == nil {
				cs.subscribeClientMutex.Unlock()

				return errors.New("subscribe service client not initialized yet")
			}
		}

		cs.subscribeClientMutex.Unlock()

		request, recvError := cs.subscribeClient.Recv()
		if recvError != nil {
			cs.subscribeClient = nil

			return cs.handleSubscribeError(ctx, recvError, "receive message from subscribe stream")
		}

		if cs.isValidRequest(ctx, request) {
			switch request.GetRequest().(type) {
			case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
				cs.queueConfigApplyRequests(ctx, request)
			default:
				cs.subscribeChannel <- request
			}
		}

		return nil
	}
}

func (cs *CommandService) handleSubscribeError(ctx context.Context, err error, errorMsg string) error {
	codeError, ok := status.FromError(err)

	if ok && codeError.Code() == codes.Unavailable {
		cs.isConnected.Store(false)
		slog.ErrorContext(ctx, fmt.Sprintf("Failed to %s, rpc unavailable. "+
			"Trying create connection rpc", errorMsg), "error", err)
		_, connectionErr := cs.CreateConnection(ctx, cs.resource)
		if connectionErr != nil {
			slog.ErrorContext(ctx, "Unable to create connection", "error", err)
		}

		return nil
	}

	slog.ErrorContext(ctx, "Failed to"+errorMsg, "error", err)

	return err
}

func (cs *CommandService) queueConfigApplyRequests(ctx context.Context, request *mpi.ManagementPlaneRequest) {
	cs.configApplyRequestQueueMutex.Lock()
	defer cs.configApplyRequestQueueMutex.Unlock()

	instanceID := request.GetConfigApplyRequest().GetOverview().GetConfigVersion().GetInstanceId()
	cs.configApplyRequestQueue[instanceID] = append(cs.configApplyRequestQueue[instanceID], request)
	if len(cs.configApplyRequestQueue[instanceID]) == 1 {
		cs.subscribeChannel <- request
	} else {
		slog.DebugContext(
			ctx,
			"Config apply request is already in progress, queuing new config apply request",
			"request", request,
		)
	}
}

func (cs *CommandService) isValidRequest(ctx context.Context, request *mpi.ManagementPlaneRequest) bool {
	var validRequest bool

	switch request.GetRequest().(type) {
	case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
		requestInstanceID := request.GetConfigApplyRequest().GetOverview().GetConfigVersion().GetInstanceId()
		validRequest = cs.checkIfInstanceExists(ctx, request, requestInstanceID)
	case *mpi.ManagementPlaneRequest_ConfigUploadRequest:
		requestInstanceID := request.GetConfigUploadRequest().GetOverview().GetConfigVersion().GetInstanceId()
		validRequest = cs.checkIfInstanceExists(ctx, request, requestInstanceID)
	case *mpi.ManagementPlaneRequest_ActionRequest:
		requestInstanceID := request.GetActionRequest().GetInstanceId()
		validRequest = cs.checkIfInstanceExists(ctx, request, requestInstanceID)
	default:
		validRequest = true
	}

	return validRequest
}

func (cs *CommandService) checkIfInstanceExists(
	ctx context.Context,
	request *mpi.ManagementPlaneRequest,
	requestInstanceID string,
) bool {
	instanceFound := false

	cs.resourceMutex.Lock()
	for _, instance := range cs.resource.GetInstances() {
		if instance.GetInstanceMeta().GetInstanceId() == requestInstanceID {
			instanceFound = true
		}
	}
	cs.resourceMutex.Unlock()

	if !instanceFound {
		slog.WarnContext(
			ctx,
			"Unable to handle request, instance not found",
			"instance", requestInstanceID,
			"request", request,
		)

		response := &mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     id.GenerateMessageID(),
				CorrelationId: request.GetMessageMeta().GetCorrelationId(),
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Unable to handle request",
				Error:   "Instance ID not found",
			},
			InstanceId: requestInstanceID,
		}
		err := cs.SendDataPlaneResponse(ctx, response)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send data plane response", "error", err)
		}
	}

	return instanceFound
}

// Retry callback for establishing the connection between the Management Plane and the Agent.
func (cs *CommandService) connectCallback(
	ctx context.Context,
	request *mpi.CreateConnectionRequest,
) func() (*mpi.CreateConnectionResponse, error) {
	return func() (*mpi.CreateConnectionResponse, error) {
		cs.subscribeClientMutex.Lock()
		response, connectErr := cs.commandServiceClient.CreateConnection(ctx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(connectErr)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to create connection", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}
}
