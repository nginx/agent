// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	GrpcClient struct {
		messagePipe bus.MessagePipeInterface
		config      *config.Config
	}
)

func NewGrpcClient(agentConfig *config.Config) *GrpcClient {
	return &GrpcClient{
		config: agentConfig,
	}
}

func getDialOptions() []grpc.DialOption {
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	return opts
}

func (grpcClient *GrpcClient) Init(messagePipe bus.MessagePipeInterface) error {
	grpcClient.messagePipe = messagePipe

	slog.Debug("Starting grpc client")
	serverAddr := net.JoinHostPort("127.0.0.1", "8080")

	conn, err := grpc.Dial(serverAddr, getDialOptions()...)
	if err != nil {
		slog.Error("error dialing %v", err)
		return nil
	}

	// nolint:all
	id, err := uuid.NewV7()
	if err != nil {
		slog.Error("error generating message id %v", err)
	}

	correlationID, err := uuid.NewUUID()
	if err != nil {
		slog.Error("error generating correlation id %v", err)
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
				InstanceId:   "1234",
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
				Version:      "v3",
			},
			InstanceConfig: &v1.InstanceConfig{},
		},
	}

	resp, err := client.CreateConnection(context.TODO(), req)
	if err != nil {
		slog.Error("error", "some", err)
	}

	slog.Debug("resp", "some", resp)

	return nil
}

func (grpcClient *GrpcClient) Close() error { return nil }

func (grpcClient *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

func (grpcClient *GrpcClient) Process(msg *bus.Message) {}

func (grpcClient *GrpcClient) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
	}
}
