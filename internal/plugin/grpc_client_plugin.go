// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"net"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	GrpcClient struct {
		logger *slog.Logger
		messagePipe bus.MessagePipeInterface
	}
)

func NewGrpcClient(agentConfig *config.Config, logger *slog.Logger) *GrpcClient {
	slog.Error("Starting grpc client")
	serverAddr := net.JoinHostPort("127.0.0.1", "8080")

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		slog.Error("error dialing %v", err)
		return nil
	}

	client := v1.NewCommandServiceClient(conn)

	req := &v1.CreateConnectionRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     0,
			CorrelationId: "",
			Timestamp:     timestamppb.Now(),
		},
		Agent:       &v1.Instance{
			InstanceMeta:   &v1.InstanceMeta{
				InstanceId:   "1234",
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
				Version:      "v3",
			},
			InstanceConfig: &v1.InstanceConfig{},
		},
	}

	resp, err := client.CreateConnection(context.TODO(), req)
	if err != nil {
		slog.Debug("error", "some", err)
	}

	slog.Debug("resp", "some", resp)

	return &GrpcClient{
		// address:              address,
		logger: logger,
	}
}

func (grpcClient *GrpcClient) Init(messagePipe bus.MessagePipeInterface) error {
	// dps.messagePipe = messagePipe
	// go dps.run(messagePipe.Context())
	return nil
}

func (grpcClient *GrpcClient) Close() error { return nil }

func (grpcClient *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "gprc-client",
	}
}

func (grpcClient *GrpcClient) Process(msg *bus.Message) {}

func (grpcClient *GrpcClient) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
	}
}
