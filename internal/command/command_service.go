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

type CommandService struct {
	commandServiceClient mpi.CommandServiceClient
	agentConfig          *config.Config
	isConnected          *atomic.Bool
	subscribeCancel      context.CancelFunc
	subscribeMutex       sync.Mutex
}

func NewCommandService(
	commandServiceClient mpi.CommandServiceClient,
	agentConfig *config.Config,
) *CommandService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &CommandService{
		commandServiceClient: commandServiceClient,
		agentConfig:          agentConfig,
		isConnected:          isConnected,
	}
}

func (cs *CommandService) UpdateDataPlaneStatus(ctx context.Context, resource *mpi.Resource) error {
	if !cs.isConnected.Load() {
		err := cs.createConnection(ctx, resource)
		if err != nil {
			return err
		}

		var subscribeCtx context.Context

		cs.subscribeMutex.Lock()
		subscribeCtx, cs.subscribeCancel = context.WithCancel(ctx)
		cs.subscribeMutex.Unlock()

		go cs.subscribe(subscribeCtx)
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

func (cs *CommandService) CancelSubscription(ctx context.Context) {
	slog.InfoContext(ctx, "Canceling subscribe context")

	cs.subscribeMutex.Lock()
	if cs.subscribeCancel != nil {
		cs.subscribeCancel()
	}
	cs.subscribeMutex.Unlock()
}

// revive cognitive complexity 13 due to the nil checks
// nolint: revive
func (cs *CommandService) subscribe(ctx context.Context) {
	var subscribeClient mpi.CommandService_SubscribeClient
	var err error

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if subscribeClient == nil {
				subscribeClient, err = cs.commandServiceClient.Subscribe(ctx)
				if err != nil {
					slog.ErrorContext(ctx, "Unable to subscribe", "error", err)
					// this is temporary and will be changed in a followup PR use back off and allow the time to be set
					// but needed to add retry to stop the logs being spammed with errors
					time.Sleep(retryInterval)

					continue
				}
			}

			request, recErr := subscribeClient.Recv()
			if recErr != nil {
				slog.ErrorContext(ctx, "Error receiving messages", "err", recErr)
				subscribeClient = nil
				time.Sleep(retryInterval)

				continue
			}

			slog.DebugContext(ctx, "Subscribe received", "request", request)

			cs.processRequest(ctx, request)
		}
	}
}

func (cs *CommandService) processRequest(ctx context.Context, request *mpi.ManagementPlaneRequest) {
	slog.InfoContext(ctx, "Management plane request not implemented yet", "request_type", request.GetRequest())
}

func (cs *CommandService) createConnection(ctx context.Context, resource *mpi.Resource) error {
	correlationID := logger.GetCorrelationID(ctx)

	request := &mpi.CreateConnectionRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	connectFn := func() (*mpi.CreateConnectionResponse, error) {
		response, connectErr := cs.commandServiceClient.CreateConnection(ctx, request)

		validatedError := grpc.ValidateGrpcError(connectErr)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to create connection", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	commonSettings := &config.CommonSettings{
		InitialInterval:     cs.agentConfig.Common.InitialInterval,
		MaxInterval:         cs.agentConfig.Common.MaxInterval,
		MaxElapsedTime:      createConnectionMaxElapsedTime,
		RandomizationFactor: cs.agentConfig.Common.RandomizationFactor,
		Multiplier:          cs.agentConfig.Common.Multiplier,
	}

	response, err := backoff.RetryWithData(connectFn, backoffHelpers.Context(ctx, commonSettings))
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "Connection created", "response", response)
	slog.DebugContext(ctx, "Agent connected")

	cs.isConnected.Store(true)

	return nil
}
