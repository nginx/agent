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

	"github.com/google/uuid"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/timestamppb"

	uuidLibrary "github.com/nginx/agent/v3/internal/uuid"
)

// these will come from the agent config
var serviceConfig = `{
		"loadBalancingPolicy": "round_robin",
		"healthCheckConfig": {
			"serviceName": "nginx-agent"
		}
	}`

type (
	GrpcClient struct {
		messagePipe bus.MessagePipeInterface
		config      *config.Config
		cancel      context.CancelFunc
		settings    *backoff.Settings
		keepAlive   keepalive.ClientParameters
	}
)

func NewGrpcClient(agentConfig *config.Config) *GrpcClient {
	if agentConfig != nil && agentConfig.Command.Server.Type == "grpc" {
		if agentConfig.Common == nil {
			slog.Error("invalid configuration settings")
			return nil
		}
		settings := &backoff.Settings{
			InitialInterval: agentConfig.Common.InitialInterval,
			MaxInterval:     agentConfig.Common.MaxInterval,
			MaxElapsedTime:  agentConfig.Common.MaxElapsedTime,
			Jitter:          agentConfig.Common.Jitter,
			Multiplier:      agentConfig.Common.Multiplier,
		}

		keepAlive := keepalive.ClientParameters{
			Time:                agentConfig.Client.Time, // add to config in future
			Timeout:             agentConfig.Client.Timeout,
			PermitWithoutStream: agentConfig.Client.PermitStream,
		}

		return &GrpcClient{
			config:    agentConfig,
			settings:  settings,
			keepAlive: keepAlive,
		}
	}

	return nil
}

func (gc *GrpcClient) Init(messagePipe bus.MessagePipeInterface) error {
	slog.Debug("Starting grpc client")
	gc.messagePipe = messagePipe

	return backoff.WaitUntil(gc.messagePipe.Context(), gc.settings, gc.createConnection)
}

func (gc *GrpcClient) createConnection() error {
	var connectionCtx context.Context

	serverAddr := net.JoinHostPort(
		gc.config.Command.Server.Host,
		fmt.Sprint(gc.config.Command.Server.Port),
	)

	connectionCtx, gc.cancel = context.WithTimeout(gc.messagePipe.Context(), gc.config.Client.Timeout)
	conn, err := grpc.DialContext(connectionCtx, serverAddr, gc.getDialOptions()...)
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

	client := v1.NewCommandServiceClient(conn)

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

	response, err := client.CreateConnection(connectionCtx, req)
	if err != nil {
		return fmt.Errorf("error creating connection: %w", err)
	}

	slog.Debug("Connection created", "response", response)

	gc.messagePipe.Process(&bus.Message{Topic: bus.GrpcConnectedTopic, Data: response})

	return nil
}

func (gc *GrpcClient) Close() error {
	if gc.cancel != nil {
		gc.cancel()
	}

	return nil
}

func (gc *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

func (gc *GrpcClient) Process(msg *bus.Message) {
	if msg.Topic == bus.GrpcConnectedTopic {
		slog.Debug("Agent connected")
	}
}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{bus.GrpcConnectedTopic}
}

func (gc *GrpcClient) getDialOptions() []grpc.DialOption {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor()),
		grpc.WithKeepaliveParams(gc.keepAlive),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	return opts
}
