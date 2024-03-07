// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"log/slog"
	"net"
	"os"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"
)

func main() {
	var grpcAddress string
	var apiAddress string

	flag.StringVar(
		&grpcAddress,
		"grpcAddress",
		"127.0.0.1:0",
		"set the gRPC address to run the server on",
	)

	flag.StringVar(
		&apiAddress,
		"apiAddress",
		"127.0.0.1:0",
		"set the API address to run the server on",
	)
	flag.Parse()

	server := mockGrpc.NewManagementGrpcServer()

	go func() {
		listener, err := net.Listen("tcp", apiAddress)
		if err != nil {
			slog.Error("Failed to create listener", "error", err)
			os.Exit(1)
		}

		server.StartServer(listener)
	}()

	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	v1.RegisterCommandServiceServer(grpcServer, server)

	slog.Info("gRPC server running", "address", listener.Addr().String())

	err = grpcServer.Serve(listener)
	if err != nil {
		slog.Error("Failed to serve server", "error", err)
		os.Exit(1)
	}
}
