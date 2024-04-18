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
		connectionMutex      sync.Mutex
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
			connectionMutex: sync.Mutex{},
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

	return backoff.WaitUntil(backOffCtx, gc.config.Common, gc.createConnectionClient)
}

func (gc *GrpcClient) createConnectionClient() error {
	gc.connectionMutex.Lock()
	defer gc.connectionMutex.Unlock()

	if gc.conn == nil || gc.conn.GetState() == connectivity.Shutdown {
		return fmt.Errorf("can't connect to server")
	}

	gc.commandServiceClient = v1.NewCommandServiceClient(gc.conn)

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
	case bus.ResourceTopic:
		if resource, ok := msg.Data.(*v1.Resource); ok {
			err := gc.createConnection(ctx, resource)
			if err != nil {
				return
			}

			err = gc.sendDataPlaneStatusUpdate(ctx, resource)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to send data plane status update", "error", err)
			}
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (gc *GrpcClient) createConnection(ctx context.Context, resource *v1.Resource) error {
	if !gc.isConnected.Load() {
		req, err := gc.createConnectionRequest(resource)
		if err != nil {
			slog.Error("Creating connection request", "error", err)
			return err
		}

		reqCtx, reqCancel := context.WithTimeout(ctx, gc.config.Common.MaxElapsedTime)
		defer reqCancel()

		response, err := gc.commandServiceClient.CreateConnection(reqCtx, req)
		if err != nil {
			slog.Error("Creating connection", "error", err)

			return err
		}

		slog.DebugContext(ctx, "Connection created", "response", response)
		slog.DebugContext(ctx, "Agent connected")

		gc.isConnected.Store(true)
	}

	return nil
}

func (gc *GrpcClient) createConnectionRequest(resource *v1.Resource) (*v1.CreateConnectionRequest, error) {
	id, err := uuid.NewV7()
	if err != nil {
		slog.Error("Generating message id", "error", err)
		return nil, err
	}

	correlationID, err := uuid.NewUUID()
	if err != nil {
		slog.Error("Generating correlation id", "error", err)
		return nil, err
	}

	return &v1.CreateConnectionRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     id.String(),
			CorrelationId: correlationID.String(),
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}, nil
}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{
		bus.GrpcConnectedTopic,
		bus.ResourceTopic,
	}
}

func (gc *GrpcClient) sendDataPlaneStatusUpdate(
	ctx context.Context,
	resource *v1.Resource,
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
		Resource: resource,
	}

	slog.DebugContext(ctx, "Sending data plane status update request", "request", request)
	if gc.commandServiceClient == nil {
		return fmt.Errorf("command service client is not initialized")
	}

	_, err := gc.commandServiceClient.UpdateDataPlaneStatus(ctx, request)

	return err
}
