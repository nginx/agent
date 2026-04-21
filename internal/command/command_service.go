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

const createConnectionMaxElapsedTime = 0

type (
	CommandService struct {
		commandServiceClient         mpi.CommandServiceClient
		subscribeClient              mpi.CommandService_SubscribeClient
		agentConfig                  *config.Config
		isConnected                  *atomic.Bool
		connectionResetInProgress    *atomic.Bool
		subscribeChannel             chan *mpi.ManagementPlaneRequest
		configApplyRequestQueue      map[string][]*mpi.ManagementPlaneRequest // key is the instance ID
		resource                     *mpi.Resource
		subscribeClientMutex         sync.Mutex
		configApplyRequestQueueMutex sync.Mutex
		resourceMutex                sync.Mutex
		agentConfigMutex             sync.RWMutex
	}
)

func NewCommandService(
	commandServiceClient mpi.CommandServiceClient,
	agentConfig *config.Config,
	subscribeChannel chan *mpi.ManagementPlaneRequest,
) *CommandService {
	return &CommandService{
		commandServiceClient:      commandServiceClient,
		agentConfig:               agentConfig,
		isConnected:               &atomic.Bool{},
		connectionResetInProgress: &atomic.Bool{},
		subscribeChannel:          subscribeChannel,
		configApplyRequestQueue:   make(map[string][]*mpi.ManagementPlaneRequest),
		resource:                  &mpi.Resource{},
	}
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

	cfg := cs.config()
	backOffCtx, backoffCancel := context.WithTimeout(ctx, cfg.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendDataPlaneStatus := func() (*mpi.UpdateDataPlaneStatusResponse, error) {
		slog.DebugContext(ctx, "Sending data plane status update request", "request", request,
			"parent_correlation_id", correlationID)

		cs.subscribeClientMutex.Lock()
		if cs.commandServiceClient == nil {
			cs.subscribeClientMutex.Unlock()
			return nil, errors.New("command service client is not initialized")
		}

		grpcCtx, cancel := context.WithTimeout(ctx, cfg.Client.Grpc.ResponseTimeout)
		defer cancel()

		response, updateError := cs.commandServiceClient.UpdateDataPlaneStatus(grpcCtx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(updateError)
		if validatedError != nil {
			slog.ErrorContext(grpcCtx, "Failed to send update data plane status", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendDataPlaneStatus,
		backoffHelpers.Context(backOffCtx, cfg.Client.Backoff),
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

	cfg := cs.config()
	backOffCtx, backoffCancel := context.WithTimeout(ctx, cfg.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	response, err := backoff.RetryWithData(
		cs.dataPlaneHealthCallback(ctx, request),
		backoffHelpers.Context(backOffCtx, cfg.Client.Backoff),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateDataPlaneHealth response", "response", response)

	return err
}

func (cs *CommandService) SendDataPlaneResponse(ctx context.Context, response *mpi.DataPlaneResponse) error {
	slog.DebugContext(ctx, "Sending data plane response", "response", response)

	cfg := cs.config()
	backOffCtx, backoffCancel := context.WithTimeout(ctx, cfg.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	err := cs.handleConfigApplyResponse(ctx, response)
	if err != nil {
		return err
	}

	return backoff.Retry(
		cs.sendDataPlaneResponseCallback(ctx, response),
		backoffHelpers.Context(backOffCtx, cfg.Client.Backoff),
	)
}

func (cs *CommandService) Reconfigure(ctx context.Context, agentConfig *config.Config) error {
	cs.agentConfigMutex.Lock()
	defer cs.agentConfigMutex.Unlock()

	slog.DebugContext(ctx, "Command plugin is reconfiguring to update agent configuration", "config", agentConfig)
	cs.agentConfig = agentConfig

	return nil
}

// Subscribe to the Management Plane for incoming commands.
func (cs *CommandService) Subscribe(ctx context.Context) {
	cfg := cs.config()
	commonSettings := &config.BackOff{
		InitialInterval:     cfg.Client.Backoff.InitialInterval,
		MaxInterval:         cfg.Client.Backoff.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cfg.Client.Backoff.RandomizationFactor,
		Multiplier:          cfg.Client.Backoff.Multiplier,
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
) (resp *mpi.CreateConnectionResponse, err error) {
	correlationID := logger.CorrelationID(ctx)
	if len(resource.GetInstances()) <= 1 {
		slog.InfoContext(ctx, "No Data Plane Instance found")
	}

	if cs.isConnected.Load() {
		return nil, errors.New("command service already connected")
	}

	if !cs.isConnected.CompareAndSwap(false, true) {
		// Another goroutine won the race and is establishing the connection.
		return nil, errors.New("command service already connected")
	}
	// If the gRPC call below fails, roll back so callers can retry.
	defer func() {
		if err != nil {
			cs.isConnected.Store(false)
		}
	}()

	request := &mpi.CreateConnectionRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	cfg := cs.config()
	commonSettings := &config.BackOff{
		InitialInterval:     cfg.Client.Backoff.InitialInterval,
		MaxInterval:         cfg.Client.Backoff.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cfg.Client.Backoff.RandomizationFactor,
		Multiplier:          cfg.Client.Backoff.Multiplier,
	}

	slog.DebugContext(ctx, "Sending create connection request", "request", request)
	resp, err = backoff.RetryWithData(
		cs.connectCallback(ctx, request),
		backoffHelpers.Context(ctx, commonSettings),
	)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Connection created", "response", resp)
	slog.InfoContext(ctx, "Agent connected")

	cs.resourceMutex.Lock()
	defer cs.resourceMutex.Unlock()
	cs.resource = resource

	return resp, nil
}

func (cs *CommandService) UpdateClient(ctx context.Context, client mpi.CommandServiceClient) error {
	cs.connectionResetInProgress.Store(true)
	defer cs.connectionResetInProgress.Store(false)

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
	// Hold the lock only long enough to search the queue, extract the requests
	// that need earlier responses, and advance the queue pointer. All I/O
	// (gRPC sends with backoff) happens after the lock is released.
	cs.configApplyRequestQueueMutex.Lock()

	instanceID := response.GetInstanceId()
	indexOfConfigApplyRequest := -1

	for index, configApplyRequest := range cs.configApplyRequestQueue[instanceID] {
		if configApplyRequest.GetMessageMeta().GetCorrelationId() == response.GetMessageMeta().GetCorrelationId() {
			indexOfConfigApplyRequest = index

			break
		}
	}

	if indexOfConfigApplyRequest < 0 {
		cs.configApplyRequestQueueMutex.Unlock()

		return nil
	}

	// Copy the requests that need a response (those ahead of the matched entry).
	requestsToRespond := make([]*mpi.ManagementPlaneRequest, indexOfConfigApplyRequest)
	copy(requestsToRespond, cs.configApplyRequestQueue[instanceID][:indexOfConfigApplyRequest])

	// Advance the queue past the matched (now handled) entry.
	cs.configApplyRequestQueue[instanceID] = cs.configApplyRequestQueue[instanceID][indexOfConfigApplyRequest+1:]
	slog.DebugContext(ctx, "Removed config apply requests from queue", "queue", cs.configApplyRequestQueue[instanceID])

	var nextPendingRequest *mpi.ManagementPlaneRequest
	if len(cs.configApplyRequestQueue[instanceID]) > 0 {
		nextPendingRequest = cs.configApplyRequestQueue[instanceID][len(cs.configApplyRequestQueue[instanceID])-1]
	}

	cs.configApplyRequestQueueMutex.Unlock()

	// Send responses for the earlier queued requests outside the lock.
	cfg := cs.config()
	for _, req := range requestsToRespond {
		newResponse := &mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     id.GenerateMessageID(),
				CorrelationId: req.GetMessageMeta().GetCorrelationId(),
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: response.GetCommandResponse(),
			InstanceId:      instanceID,
			RequestType:     response.GetRequestType(),
		}

		slog.DebugContext(ctx, "Sending data plane response for queued config apply request", "response", newResponse)

		backOffCtx, backoffCancel := context.WithTimeout(ctx, cfg.Client.Backoff.MaxElapsedTime)

		err := backoff.Retry(
			cs.sendDataPlaneResponseCallback(ctx, newResponse),
			backoffHelpers.Context(backOffCtx, cfg.Client.Backoff),
		)
		backoffCancel()

		if err != nil {
			slog.ErrorContext(ctx, "Failed to send data plane response", "error", err)

			return err
		}
	}

	if nextPendingRequest != nil && !cs.connectionResetInProgress.Load() {
		cs.subscribeChannel <- nextPendingRequest
	}

	return nil
}

// Retry callback for sending a data plane health status to the Management Plane.
func (cs *CommandService) dataPlaneHealthCallback(
	ctx context.Context,
	request *mpi.UpdateDataPlaneHealthRequest,
) func() (*mpi.UpdateDataPlaneHealthResponse, error) {
	cfg := cs.config()
	return func() (*mpi.UpdateDataPlaneHealthResponse, error) {
		slog.DebugContext(ctx, "Sending data plane health update request", "request", request)

		cs.subscribeClientMutex.Lock()
		if cs.commandServiceClient == nil {
			cs.subscribeClientMutex.Unlock()
			return nil, errors.New("command service client is not initialized")
		}

		grpcCtx, cancel := context.WithTimeout(ctx, cfg.Client.Grpc.ResponseTimeout)
		defer cancel()

		response, updateError := cs.commandServiceClient.UpdateDataPlaneHealth(grpcCtx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(grpcCtx, "Failed to send update data plane health", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}
}

// Retry callback for receiving messages from the Management Plane subscription.
//
//nolint:revive // cognitive complexity is 18
func (cs *CommandService) receiveCallback(ctx context.Context) func() error {
	return func() error {
		if cs.connectionResetInProgress.Load() {
			slog.DebugContext(ctx, "Connection reset in progress, skipping receive from subscribe stream")

			return nil
		}

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
			cs.subscribeClientMutex.Lock()
			cs.subscribeClient = nil
			cs.subscribeClientMutex.Unlock()

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
	cs.isConnected.Store(false)

	slog.ErrorContext(ctx, fmt.Sprintf("Failed to %s. "+
		"Trying create connection rpc again", errorMsg), "error", err)

	_, connectionErr := cs.CreateConnection(ctx, cs.resource)
	if connectionErr != nil {
		slog.ErrorContext(ctx, "Unable to create connection", "error", connectionErr)
	}

	return err
}

func (cs *CommandService) queueConfigApplyRequests(ctx context.Context, request *mpi.ManagementPlaneRequest) {
	cs.configApplyRequestQueueMutex.Lock()

	instanceID := request.GetConfigApplyRequest().GetOverview().GetConfigVersion().GetInstanceId()
	cs.configApplyRequestQueue[instanceID] = append(cs.configApplyRequestQueue[instanceID], request)
	shouldSend := len(cs.configApplyRequestQueue[instanceID]) == 1 && !cs.connectionResetInProgress.Load()
	cs.configApplyRequestQueueMutex.Unlock()

	if shouldSend {
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
	cfg := cs.config()
	return func() (*mpi.CreateConnectionResponse, error) {
		grpcCtx, cancel := context.WithTimeout(ctx, cfg.Client.Grpc.ResponseTimeout)
		defer cancel()

		cs.subscribeClientMutex.Lock()
		response, connectErr := cs.commandServiceClient.CreateConnection(grpcCtx, request)
		cs.subscribeClientMutex.Unlock()

		validatedError := grpc.ValidateGrpcError(connectErr)
		if validatedError != nil {
			slog.ErrorContext(grpcCtx, "Failed to create connection", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}
}

func (cs *CommandService) config() *config.Config {
	cs.agentConfigMutex.RLock()
	defer cs.agentConfigMutex.RUnlock()

	return cs.agentConfig
}
