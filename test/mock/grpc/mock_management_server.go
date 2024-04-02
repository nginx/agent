// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcvalidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	ServiceName           = "mock-management-plane"
	maxConnectionIdle     = 15 * time.Second
	maxConnectionAge      = 30 * time.Second
	maxConnectionAgeGrace = 5 * time.Second
	keepAliveTime         = 5 * time.Second
	keepAliveTimeout      = 1 * time.Second
)

var keepAliveEnforcementPolicy = keepalive.EnforcementPolicy{
	MinTime:             keepAliveTime,
	PermitWithoutStream: true,
}

var keepAliveServerParameters = keepalive.ServerParameters{
	MaxConnectionIdle:     maxConnectionIdle,
	MaxConnectionAge:      maxConnectionAge,
	MaxConnectionAgeGrace: maxConnectionAgeGrace,
	Time:                  keepAliveTime,
	Timeout:               keepAliveTimeout,
}

type MockManagementServer struct {
	CommandServer *CommandServer
	FileServer    *FileServer
	GrpcServer    *grpc.Server
}

func NewMockManagementServer(
	apiAddress, grpcAddress, configDirectory string,
	sleepDuration *time.Duration,
) *MockManagementServer {
	commandServer := NewCommandServer()

	go func() {
		listener, listenError := net.Listen("tcp", apiAddress)
		if listenError != nil {
			slog.Error("Failed to create listener", "error", listenError)
		}

		commandServer.StartServer(listener)
	}()

	fileServer, err := NewFileServer(configDirectory)
	if err != nil {
		slog.Error("Failed to create file server", "error", err)
	}

	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("Failed to listen", "error", err)
	}
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(grpcvalidator.UnaryServerInterceptor(), ensureValidToken),
		grpc.StreamInterceptor(grpcvalidator.StreamServerInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepAliveEnforcementPolicy),
		grpc.KeepaliveParams(keepAliveServerParameters),
	}

	grpcServer := grpc.NewServer(opts...)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)

	v1.RegisterCommandServiceServer(grpcServer, commandServer)
	v1.RegisterFileServiceServer(grpcServer, fileServer)
	go reportHealth(healthcheck, *sleepDuration)

	go func() {
		slog.Info("gRPC server running", "address", listener.Addr().String())

		err := grpcServer.Serve(listener)
		if err != nil {
			slog.Error("Failed to serve server", "error", err)
		}
	}()

	return &MockManagementServer{
		CommandServer: commandServer,
		FileServer:    fileServer,
		GrpcServer:    grpcServer,
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

func ensureValidToken(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	var (
		errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
		errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid token")
	)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMissingMetadata
	}
	// The keys within metadata.MD are normalized to lowercase.
	// See: https://godoc.org/google.golang.org/grpc/metadata#New
	if !valid(md["authorization"]) {
		return nil, errInvalidToken
	}
	// Continue execution of handler after ensuring a valid token.
	return handler(ctx, req)
}

// valid validates the authorization.
func valid(authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	// Perform the token validation here. For the sake of this example, the code
	// here forgoes any of the usual OAuth2 token validation and instead checks
	// for a token matching an arbitrary string.
	return token == "1234"
}
