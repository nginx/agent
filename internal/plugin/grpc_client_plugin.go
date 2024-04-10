// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentGrpc "github.com/nginx/agent/v3/internal/grpc"
)

type (
	GrpcClient struct {
		messagePipe          bus.MessagePipeInterface
		config               *config.Config
		conn                 *grpc.ClientConn
		isConnected          *atomic.Bool
		commandServiceClient v1.CommandServiceClient
		cancel               context.CancelFunc
		instances            []*v1.Instance
		connectionMutex      sync.Mutex
		instancesMutex       sync.Mutex
	}
)

func NewGrpcClient(agentConfig *config.Config) *GrpcClient {
	if agentConfig != nil && agentConfig.Command.Server.Type == "grpc" {
		if agentConfig.Common == nil {
			slog.Error("Invalid common configuration settings")
			return nil
		}

		isConnected := &atomic.Bool{}
		isConnected.Store(false)

		return &GrpcClient{
			config:          agentConfig,
			isConnected:     isConnected,
			instances:       []*v1.Instance{},
			connectionMutex: sync.Mutex{},
			instancesMutex:  sync.Mutex{},
		}
	}

	return nil
}

func (gc *GrpcClient) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	var (
		grpcClientCtx context.Context
		err           error
	)

	slog.InfoContext(ctx, "Starting grpc client plugin")
	gc.messagePipe = messagePipe

	serverAddr := net.JoinHostPort(
		gc.config.Command.Server.Host,
		fmt.Sprint(gc.config.Command.Server.Port),
	)

	grpcClientCtx, gc.cancel = context.WithTimeout(ctx, gc.config.Client.Timeout)
	slog.InfoContext(ctx, "Dialing grpc server", "server_addr", serverAddr)

	gc.connectionMutex.Lock()
	gc.conn, err = grpc.DialContext(grpcClientCtx, serverAddr, agentGrpc.GetDialOptions(gc.config)...)
	gc.connectionMutex.Unlock()

	if err != nil {
		return err
	}
	backOffCtx, backoffCancel := context.WithTimeout(ctx, gc.config.Client.Timeout)

	defer backoffCancel()

	return backoff.WaitUntil(backOffCtx, gc.config.Common, gc.createConnection)
}

func (gc *GrpcClient) createConnection() error {
	// nolint: revive
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("error generating message id: %w", err)
	}

	correlationID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("error generating correlation id: %w", err)
	}

	gc.connectionMutex.Lock()
	defer gc.connectionMutex.Unlock()

	if gc.conn == nil || gc.conn.GetState() == connectivity.Shutdown {
		return fmt.Errorf("can't connect to server")
	}

	gc.commandServiceClient = v1.NewCommandServiceClient(gc.conn)
	req := &v1.CreateConnectionRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     id.String(),
			CorrelationId: correlationID.String(),
			Timestamp:     timestamppb.Now(),
		},
		Agent: &v1.Instance{
			InstanceMeta: &v1.InstanceMeta{
				InstanceId:   agentGrpc.UUID,
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
				Version:      gc.config.Version,
			},
			InstanceConfig: &v1.InstanceConfig{},
		},
	}

	ctx := context.Background()
	reqCtx, reqCancel := context.WithTimeout(ctx, gc.config.Common.MaxElapsedTime)
	defer reqCancel()

	response, err := gc.commandServiceClient.CreateConnection(reqCtx, req)
	if err != nil {
		return fmt.Errorf("creating connection: %w", err)
	}

	slog.Debug("Connection created", "response", response)
	gc.messagePipe.Process(ctx, &bus.Message{Topic: bus.GrpcConnectedTopic, Data: response})

	return nil
}

func (gc *GrpcClient) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing grpc client plugin")

	gc.connectionMutex.Lock()
	defer gc.connectionMutex.Unlock()

	if gc.conn != nil {
		err := gc.conn.Close()
		if err != nil && gc.cancel != nil {
			slog.ErrorContext(ctx, "Failed to gracefully close gRPC connection", "error", err)
			gc.cancel()
		}
	}

	return nil
}

func (gc *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

func (gc *GrpcClient) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
			gc.instancesMutex.Lock()
			gc.instances = newInstances
			gc.instancesMutex.Unlock()

			err := gc.sendDataPlaneStatusUpdate(ctx, newInstances)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to send data plane status update", "error", err)
			}
		}
	case bus.GrpcConnectedTopic:
		slog.DebugContext(ctx, "Agent connected")
		gc.isConnected.Store(true)

		gc.instancesMutex.Lock()
		err := gc.sendDataPlaneStatusUpdate(ctx, gc.instances)
		gc.instancesMutex.Unlock()

		if err != nil {
			slog.ErrorContext(ctx, "Unable to send data plane status update", "error", err)
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
		bus.GrpcConnectedTopic,
	}
}

func (gc *GrpcClient) sendDataPlaneStatusUpdate(
	ctx context.Context,
	instances []*v1.Instance,
) error {
	if !gc.isConnected.Load() {
		slog.DebugContext(ctx, "gRPC client not connected yet. Skipping sending data plane status update")
		return nil
	}

	correlationID := logger.GetCorrelationID(ctx)

	request := &v1.UpdateDataPlaneStatusRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Instances: instances,
	}

	slog.DebugContext(ctx, "Sending data plane status update request", "request", request)
	if gc.commandServiceClient == nil {
		return fmt.Errorf("command service client is not initialized")
	}

	_, err := gc.commandServiceClient.UpdateDataPlaneStatus(ctx, request)

	return err
}
