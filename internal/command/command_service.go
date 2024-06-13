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
		commandServiceClient mpi.CommandServiceClient
		agentConfig          *config.Config
		isConnected          *atomic.Bool
		subscribeCancel      context.CancelFunc
		subscribeMutex       sync.Mutex
		subscribeChannel     chan *mpi.ManagementPlaneRequest
		subscribeClient      mpi.CommandService_SubscribeClient
		subscribeClientMutex sync.Mutex
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
		commandServiceClient: commandServiceClient,
		agentConfig:          agentConfig,
		isConnected:          isConnected,
		subscribeChannel:     subscribeChannel,
	}

	var subscribeCtx context.Context

	commandService.subscribeMutex.Lock()
	subscribeCtx, commandService.subscribeCancel = context.WithCancel(ctx)
	commandService.subscribeMutex.Unlock()

	go commandService.subscribe(subscribeCtx)

	return commandService
}

func (cs *CommandService) UpdateDataPlaneStatus(ctx context.Context, resource *mpi.Resource) error {
	if !cs.isConnected.Load() {
		err := cs.createConnection(ctx, resource)
		if err != nil {
			return err
		}
	}

	correlationID := logger.GetCorrelationID(ctx)

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
		slog.DebugContext(ctx, "Sending data plane status update request", "request", request)
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

	sendDataPlaneHealth := func() (*mpi.UpdateDataPlaneHealthResponse, error) {
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

	response, err := backoff.RetryWithData(
		sendDataPlaneHealth,
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

	sendDataPlaneResponse := func() error {
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

	return backoff.Retry(
		sendDataPlaneResponse,
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

// nolint: revive,gocognit
func (cs *CommandService) subscribe(ctx context.Context) {
	commonSettings := &config.CommonSettings{
		InitialInterval:     cs.agentConfig.Common.InitialInterval,
		MaxInterval:         cs.agentConfig.Common.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Common.RandomizationFactor,
		Multiplier:          cs.agentConfig.Common.Multiplier,
	}

	err := backoff.Retry(cs.subscribeCallback(ctx), backoffHelpers.Context(ctx, commonSettings))
	if err != nil {
		slog.WarnContext(ctx, "Failed to initialize the subscribe stream", "error", err)
		return
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

// Retry callback for initializing the subscription to the Management Plane.
func (cs *CommandService) subscribeCallback(ctx context.Context) func() error {
	return func() error {
		cs.subscribeClientMutex.Lock()
		defer cs.subscribeClientMutex.Unlock()

		if cs.subscribeClient == nil {
			if cs.commandServiceClient == nil {
				return errors.New("command service client is not initialized")
			}

			var err error
			cs.subscribeClient, err = cs.commandServiceClient.Subscribe(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to create subscribe client", "error", err)

				return err
			}
		}

		return nil
	}
}

// Retry callback for receiving messages from the Management Plane subscription.
func (cs *CommandService) receiveCallback(ctx context.Context) func() error {
	return func() error {
		request, recvError := cs.subscribeClient.Recv()
		if recvError != nil {
			slog.ErrorContext(ctx, "Failed to receive message from subscribe stream", "error", recvError)
			cs.subscribeClient = nil

			return recvError
		}

		cs.subscribeChannel <- request

		return nil
	}
}

func (cs *CommandService) createConnection(ctx context.Context, resource *mpi.Resource) error {
	correlationID := logger.GetCorrelationID(ctx)

	// Only send a resource update message if instances other than the agent instance are found
	if len(resource.GetInstances()) <= 1 {
		return errors.New("waiting for data plane instances to be found before sending create connection request")
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

	response, err := backoff.RetryWithData(
		cs.connectCallback(ctx, request),
		backoffHelpers.Context(ctx, commonSettings),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "Connection created", "response", response)
	slog.DebugContext(ctx, "Agent connected")

	cs.isConnected.Store(true)

	return nil
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
