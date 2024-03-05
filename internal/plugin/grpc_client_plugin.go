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
)

type (
	GrpcClient struct {
		// address              string
		logger               *slog.Logger
		// instances            []*instances.Instance
		messagePipe          bus.MessagePipeInterface
		// server               net.Listener
	}
)

func NewGrpcClient(agentConfig *config.Config, logger *slog.Logger) *GrpcClient {
	serverAddr := net.JoinHostPort("127.0.0.1", "8080")

	// var opts []grpc.DialOption

	conn, err := grpc.Dial(serverAddr)
	if (err != nil) {
		return nil
	}

	client := v1.NewCommandServiceClient(conn)

	req := &v1.CreateConnectionRequest{}

	resp, err := client.CreateConnection(context.TODO(), req)

	slog.Debug("%v", resp)

	return &GrpcClient{
		// address:              address,
		logger:               logger,
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
