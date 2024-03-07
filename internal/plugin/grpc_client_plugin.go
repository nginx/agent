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
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	uuidLibrary "github.com/nginx/agent/v3/internal/uuid"
)

type (
	GrpcClient struct {
		messagePipe bus.MessagePipeInterface
		config      *config.Config
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

func (gc *GrpcClient) Init(messagePipe bus.MessagePipeInterface) error {
	slog.Debug("Starting grpc client")
	gc.messagePipe = messagePipe

	serverAddr := net.JoinHostPort(
		gc.config.Command.Server.Host,
		fmt.Sprint(gc.config.Command.Server.Port),
	)

	conn, err := grpc.Dial(serverAddr, getDialOptions()...)
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

	response, err := client.CreateConnection(context.TODO(), req)
	if err != nil {
		return fmt.Errorf("error creating connection: %w", err)
	}

	slog.Debug("Connection created", "response", response)

	return nil
}

func (gc *GrpcClient) Close() error { return nil }

func (gc *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

func (gc *GrpcClient) Process(msg *bus.Message) {}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{}
}

func getDialOptions() []grpc.DialOption {
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	return opts
}
