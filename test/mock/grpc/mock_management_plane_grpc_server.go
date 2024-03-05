// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc"
)

type ManagementGrpcServer struct {
}

func NewManagementGrpcServer(configDirectory string) *ManagementGrpcServer {
	ms := &ManagementGrpcServer{}

	return ms
}

func (s *ManagementGrpcServer) CreateConnection(ctx context.Context, in *v1.CreateConnectionRequest, opts ...grpc.CallOption) (*v1.CreateConnectionResponse, error) {
	return nil, nil
}

func (s *ManagementGrpcServer) UpdateDataPlaneStatus(ctx context.Context, in *v1.UpdateDataPlaneStatusRequest, opts ...grpc.CallOption) (*v1.UpdateDataPlaneStatusResponse, error) {
	return nil, nil
}

func (s *ManagementGrpcServer) UpdateDataPlaneHealth(ctx context.Context, in *v1.UpdateDataPlaneHealthRequest, opts ...grpc.CallOption) (*v1.UpdateDataPlaneHealthResponse, error) {
	return nil, nil
}

func (s *ManagementGrpcServer) Subscribe(ctx context.Context, opts ...grpc.CallOption) (v1.CommandService_SubscribeClient, error) {
	return nil, nil
}

