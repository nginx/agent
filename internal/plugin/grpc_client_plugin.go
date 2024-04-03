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
	"time"

	"github.com/google/uuid"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/timestamppb"

	uuidLibrary "github.com/nginx/agent/v3/internal/uuid"
)

const (
	DefaultClientTime   = 120  // Default time for client operations in seconds
	DefaultTimeout      = 60   // Default timeout duration in seconds
	DefaultPermitStream = true // Flag indicating permission of stream
)

type (
	GrpcClient struct {
		messagePipe          bus.MessagePipeInterface
		config               *config.Config
		conn                 *grpc.ClientConn
		commandServiceClient v1.CommandServiceClient
		cancel               context.CancelFunc
	}
)

func NewGrpcClient(agentConfig *config.Config) *GrpcClient {
	if agentConfig != nil && agentConfig.Command.Server.Type == "grpc" {
		return &GrpcClient{
			config: agentConfig,
		}
	}

	return nil
}

func (gc *GrpcClient) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.Debug("Starting grpc client plugin")
	gc.messagePipe = messagePipe

	serverAddr := net.JoinHostPort(
		gc.config.Command.Server.Host,
		fmt.Sprint(gc.config.Command.Server.Port),
	)

	var grpcClientCtx context.Context
	grpcClientCtx, gc.cancel = context.WithTimeout(ctx, gc.config.Client.Timeout)

	var err error
	gc.conn, err = grpc.DialContext(grpcClientCtx, serverAddr, gc.getDialOptions()...)
	if err != nil {
		return err
	}

	// nolint: revive
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("error generating message id: %w", err)
	}

	correlationID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("error generating correlation id: %w", err)
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
				InstanceId:   uuidLibrary.Generate("/etc/nginx-agent/nginx-agent"),
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
				Version:      gc.config.Version,
			},
			InstanceConfig: &v1.InstanceConfig{},
		},
	}

	response, err := gc.commandServiceClient.CreateConnection(grpcClientCtx, req)
	if err != nil {
		return fmt.Errorf("error creating connection: %w", err)
	}

	slog.Debug("Connection created", "response", response)

	return nil
}

func (gc *GrpcClient) Close(_ context.Context) error {
	slog.Debug("Closing grpc client plugin")

	err := gc.conn.Close()
	if err != nil {
		slog.Error("Failed to gracefully close gRPC connection", "error", err)
		gc.cancel()
	}

	return nil
}

func (gc *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

func (gc *GrpcClient) Process(ctx context.Context, msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
			err := gc.sendDataPlaneStatusUpdate(ctx, newInstances)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to send data plane status update", "error", err)
			}
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
	}
}

func (gc *GrpcClient) getDialOptions() []grpc.DialOption {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                DefaultClientTime * time.Second, // add to config in future
			Timeout:             DefaultTimeout * time.Second,
			PermitWithoutStream: DefaultPermitStream,
		}),
	}

	return opts
}

func (gc *GrpcClient) sendDataPlaneStatusUpdate(
	ctx context.Context,
	instances []*v1.Instance,
) error {
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
	_, err := gc.commandServiceClient.UpdateDataPlaneStatus(ctx, request)

	return err
}
