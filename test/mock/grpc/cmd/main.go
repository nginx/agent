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
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	DefaultSleepDuration = time.Millisecond * 100
)

func main() {
	configDirectory, grpcAddress, apiAddress, system, sleep, err := getFlags()

	commandServer := mockGrpc.NewManagementGrpcServer()

	go func() {
		listener, listenError := net.Listen("tcp", apiAddress)
		if listenError != nil {
			slog.Error("Failed to create listener", "error", err)
			os.Exit(1)
		}

		commandServer.StartServer(listener)
	}()

	fileServer, err := mockGrpc.NewManagementGrpcFileServer(configDirectory)
	if err != nil {
		slog.Error("Failed to create file server: %v", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("Failed to listen: %v", err)
		os.Exit(1)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)

	v1.RegisterCommandServiceServer(grpcServer, commandServer)
	v1.RegisterFileServiceServer(grpcServer, fileServer)

	slog.Info("gRPC server running", "address", listener.Addr().String())

	// asynchronously inspect dependencies and toggle serving status as needed
	// temporarily set to not serving for a chaos test
	// flip back to a healthy state
	go reportHealth(healthcheck, system, sleep)

	err = grpcServer.Serve(listener)
	if err != nil {
		slog.Error("Failed to serve server", "error", err)
		os.Exit(1)
	}
}

func reportHealth(healthcheck *health.Server, system string, sleep time.Duration) {
	next := healthgrpc.HealthCheckResponse_SERVING
	for {
		healthcheck.SetServingStatus(system, next)

		if next == healthgrpc.HealthCheckResponse_SERVING && (time.Now().Unix()%32 == 0) {
			next = healthgrpc.HealthCheckResponse_NOT_SERVING
		} else if next == healthgrpc.HealthCheckResponse_NOT_SERVING {
			next = healthgrpc.HealthCheckResponse_SERVING
		}
		time.Sleep(sleep)
	}
}

func getFlags() (configDirectory, grpcAddress, apiAddress, system string, sleep time.Duration, err error) {
	currentPath, err := os.Getwd()
	if err != nil {
		slog.Error("Unable to get current directory", "error", err)
	}

	flag.Duration(
		"sleep",
		DefaultSleepDuration,
		"duration between changes in health",
	)

	flag.StringVar(
		&configDirectory,
		"configDirectory",
		currentPath,
		"set the directory where the config files are stored",
	)

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

	return configDirectory, grpcAddress, apiAddress, system, sleep, err
}
