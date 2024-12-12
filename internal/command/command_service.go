// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
)

var _ commandService = (*CommandService)(nil)

const (
	retryInterval                  = 5 * time.Second
	createConnectionMaxElapsedTime = 0
)

type (
	CommandService struct {
		commandServiceClient         mpi.CommandServiceClient
		subscribeClient              mpi.CommandService_SubscribeClient
		agentConfig                  *config.Config
		isConnected                  *atomic.Bool
		subscribeCancel              context.CancelFunc
		subscribeChannel             chan *mpi.ManagementPlaneRequest
		configApplyRequestQueue      map[string][]*mpi.ManagementPlaneRequest // key is the instance ID
		subscribeMutex               sync.Mutex
		subscribeClientMutex         sync.Mutex
		configApplyRequestQueueMutex sync.Mutex
	}
)

func NewCommandService(
	ctx context.Context,
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
	}

	var subscribeCtx context.Context

	commandService.subscribeMutex.Lock()
	subscribeCtx, commandService.subscribeCancel = context.WithCancel(ctx)
	commandService.subscribeMutex.Unlock()

	go commandService.subscribe(subscribeCtx)

	return commandService
}

func (cs *CommandService) IsConnected() bool {
	return cs.isConnected.Load()
}

func (cs *CommandService) UpdateDataPlaneStatus(
	ctx context.Context,
	resource *mpi.Resource,
) error {
	correlationID := logger.GetCorrelationID(ctx)
	if !cs.isConnected.Load() {
		return errors.New("command service client not connected yet")
	}

	request := &mpi.UpdateDataPlaneStatusRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Common.MaxElapsedTime)
	defer backoffCancel()

	sendDataPlaneStatus := func() (*mpi.UpdateDataPlaneStatusResponse, error) {
		slog.DebugContext(ctx, "Sending data plane status update request", "request", request,
			"parent_correlation_id", correlationID)
		if cs.commandServiceClient == nil {
			return nil, errors.New("command service client is not initialized")
		}

		response, updateError := cs.commandServiceClient.UpdateDataPlaneStatus(ctx, request)

		validatedError := grpc.ValidateGrpcError(updateError)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update data plane status", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendDataPlaneStatus,
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Common),
	)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "UpdateDataPlaneStatus response", "response", response)

	return err
}

func (cs *CommandService) UpdateDataPlaneHealth(ctx context.Context, instanceHealths []*mpi.InstanceHealth) error {
	if !cs.isConnected.Load() {
		return errors.New("command service client not connected yet")
	}

	correlationID := logger.GetCorrelationID(ctx)

	request := &mpi.UpdateDataPlaneHealthRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		InstanceHealths: instanceHealths,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Common.MaxElapsedTime)
	defer backoffCancel()

	response, err := backoff.RetryWithData(
		cs.dataPlaneHealthCallback(ctx, request),
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Common),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateDataPlaneHealth response", "response", response)

	return err
}

func (cs *CommandService) SendDataPlaneResponse(ctx context.Context, response *mpi.DataPlaneResponse) error {
	slog.DebugContext(ctx, "Sending data plane response", "response", response)

	backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Common.MaxElapsedTime)
	defer backoffCancel()

	err := cs.handleConfigApplyResponse(ctx, response)
	if err != nil {
		return err
	}

	return backoff.Retry(
		cs.sendDataPlaneResponseCallback(ctx, response),
		backoffHelpers.Context(backOffCtx, cs.agentConfig.Common),
	)
}

func (cs *CommandService) CancelSubscription(ctx context.Context) {
	slog.InfoContext(ctx, "Canceling subscribe context")

	cs.subscribeMutex.Lock()
	if cs.subscribeCancel != nil {
		cs.subscribeCancel()
	}
	cs.subscribeMutex.Unlock()
}

func (cs *CommandService) subscribe(ctx context.Context) {
	commonSettings := &config.CommonSettings{
		InitialInterval:     cs.agentConfig.Common.InitialInterval,
		MaxInterval:         cs.agentConfig.Common.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Common.RandomizationFactor,
		Multiplier:          cs.agentConfig.Common.Multiplier,
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
	correlationID := logger.GetCorrelationID(ctx)
	if len(resource.GetInstances()) <= 1 {
		slog.InfoContext(ctx, "No Data Plane Instance found")
	}

	request := &mpi.CreateConnectionRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	commonSettings := &config.CommonSettings{
		InitialInterval:     cs.agentConfig.Common.InitialInterval,
		MaxInterval:         cs.agentConfig.Common.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Common.RandomizationFactor,
		Multiplier:          cs.agentConfig.Common.Multiplier,
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

	return response, nil
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

	// TODO: fix this
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
	for i := 0; i < indexOfConfigApplyRequest; i++ {
		newResponse := response

		newMessageID, err := uuid.NewV7()
		if err != nil {
			slog.DebugContext(ctx, "Failed to create new message ID", "error", err)
		} else {
			newResponse.GetMessageMeta().MessageId = newMessageID.String()
		}

		request := cs.configApplyRequestQueue[instanceID][i]
		newResponse.GetMessageMeta().CorrelationId = request.GetMessageMeta().GetCorrelationId()

		slog.DebugContext(
			ctx,
			"Sending data plane response for queued config apply request",
			"response", newResponse,
		)

		backOffCtx, backoffCancel := context.WithTimeout(ctx, cs.agentConfig.Common.MaxElapsedTime)

		err = backoff.Retry(
			cs.sendDataPlaneResponseCallback(ctx, newResponse),
			backoffHelpers.Context(backOffCtx, cs.agentConfig.Common),
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
		if cs.commandServiceClient == nil {
			return nil, errors.New("command service client is not initialized")
		}

		response, updateError := cs.commandServiceClient.UpdateDataPlaneHealth(ctx, request)

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
				slog.ErrorContext(ctx, "Failed to create subscribe client", "error", err)
				cs.subscribeClientMutex.Unlock()

				return err
			}

			if cs.subscribeClient == nil {
				cs.subscribeClientMutex.Unlock()
				return errors.New("subscribe service client not initialized yet")
			}
		}

		cs.subscribeClientMutex.Unlock()

		request, recvError := cs.subscribeClient.Recv()
		if recvError != nil {
			slog.ErrorContext(ctx, "Failed to receive message from subscribe stream", "error", recvError)
			cs.subscribeClient = nil

			return recvError
		}

		switch request.GetRequest().(type) {
		case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
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
		default:
			cs.subscribeChannel <- request
		}

		return nil
	}
}

// Retry callback for establishing the connection between the Management Plane and the Agent.
func (cs *CommandService) connectCallback(
	ctx context.Context,
	request *mpi.CreateConnectionRequest,
) func() (*mpi.CreateConnectionResponse, error) {
	return func() (*mpi.CreateConnectionResponse, error) {
		response, connectErr := cs.commandServiceClient.CreateConnection(ctx, request)

		validatedError := grpc.ValidateGrpcError(connectErr)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to create connection", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}
}
