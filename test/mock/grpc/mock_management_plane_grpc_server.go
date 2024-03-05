// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type ManagementGrpcServer struct {
	v1.UnimplementedCommandServiceServer
}

func NewManagementGrpcServer() *ManagementGrpcServer {
	ms := &ManagementGrpcServer{}

	return ms
}

func (s *ManagementGrpcServer) CreateConnection(ctx context.Context, in *v1.CreateConnectionRequest) (*v1.CreateConnectionResponse, error) {
	slog.Debug("hit create connection")

	return &v1.CreateConnectionResponse{
		Response: &v1.CommandResponse{
			Status:  v1.CommandResponse_COMMAND_STATUS_OK,
			Message: "Success",
		},
	}, nil
}

func (s *ManagementGrpcServer) UpdateDataPlaneStatus(ctx context.Context, in *v1.UpdateDataPlaneStatusRequest) (*v1.UpdateDataPlaneStatusResponse, error) {
	return nil, nil
}

func (s *ManagementGrpcServer) UpdateDataPlaneHealth(ctx context.Context, in *v1.UpdateDataPlaneHealthRequest) (*v1.UpdateDataPlaneHealthResponse, error) {
	return nil, nil
}

func (s *ManagementGrpcServer) Subscribe(in v1.CommandService_SubscribeServer) error {
	return nil
}
