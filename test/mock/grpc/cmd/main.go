// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/logger"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	DefaultSleepDuration = time.Millisecond * 100
	ServiceName          = "mock-management-plane"
	filePermissions      = 0o640
)

var (
	sleepDuration   = flag.Duration("sleepDuration", DefaultSleepDuration, "duration between changes in health")
	configDirectory = flag.String("configDirectory", "", "set the directory where the config files are stored")
	// address         = flag.String("address", "127.0.0.1:0", "set the address to run the server on")
	grpcAddress = flag.String("grpcAddress", "127.0.0.1:0", "set the gRPC address to run the server on")
	apiAddress  = flag.String("apiAddress", "127.0.0.1:0", "set the API address to run the server on")
	logLevel    = flag.String("logLevel", "INFO", "set the log level")
)

func main() {
	flag.Parse()

	newLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logger.GetLogLevel(*logLevel),
	}))
	slog.SetDefault(newLogger)

	if configDirectory == nil {
		defaultConfigDirectory, err := generateDefaultConfigDirectory()
		configDirectory = &defaultConfigDirectory
		if err != nil {
			slog.Error("Failed to create default config directory", "error", err)
			os.Exit(1)
		}
	}

	commandServer := mockGrpc.NewManagementGrpcServer()

	go func() {
		listener, listenError := net.Listen("tcp", *apiAddress)
		if listenError != nil {
			slog.Error("Failed to create listener", "error", listenError)
			os.Exit(1)
		}

		commandServer.StartServer(listener)
	}()

	fileServer, err := mockGrpc.NewManagementGrpcFileServer(*configDirectory)
	if err != nil {
		slog.Error("Failed to create file server", "error", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", *grpcAddress)
	if err != nil {
		slog.Error("Failed to listen", "error", err)
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
	go reportHealth(healthcheck, *sleepDuration)

	err = grpcServer.Serve(listener)
	if err != nil {
		slog.Error("Failed to serve server", "error", err)
		os.Exit(1)
	}
}

func reportHealth(healthcheck *health.Server, sleep time.Duration) {
	next := healthgrpc.HealthCheckResponse_SERVING
	for {
		healthcheck.SetServingStatus(ServiceName, next)

		if next == healthgrpc.HealthCheckResponse_SERVING && (time.Now().Unix()%32 == 0) {
			next = healthgrpc.HealthCheckResponse_NOT_SERVING
		} else if next == healthgrpc.HealthCheckResponse_NOT_SERVING {
			next = healthgrpc.HealthCheckResponse_SERVING
		}
		time.Sleep(sleep)
	}
}

func generateDefaultConfigDirectory() (string, error) {
	tempDirectory := os.TempDir()

	err := os.MkdirAll(filepath.Join(tempDirectory, "config/1/etc/nginx"), filePermissions)
	if err != nil {
		slog.Error("Failed to create directories", "error", err)
		return "", err
	}

	source, err := os.Open("test/config/nginx/nginx.conf")
	if err != nil {
		slog.Error("Failed to open nginx.conf", "error", err)
		return "", err
	}
	defer source.Close()

	destination, err := os.Create(filepath.Join(tempDirectory, "config/1/etc/nginx/nginx.conf"))
	if err != nil {
		slog.Error("Failed to create nginx.conf", "error", err)
		return "", err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		slog.Error("Failed to copy nginx.conf", "error", err)
		return "", err
	}

	return filepath.Join(tempDirectory, "config"), nil
}
