// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"
)

func main() {
	server := mockGrpc.NewManagementGrpcServer()

	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", "8080"))
	if err != nil {
		slog.Error("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	v1.RegisterCommandServiceServer(grpcServer, server)
	grpcServer.Serve(lis)
}
