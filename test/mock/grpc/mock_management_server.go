// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/nginx/agent/v3/internal/config"

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
	maxConnectionIdle     = 15 * time.Millisecond
	maxConnectionAge      = 30 * time.Millisecond
	maxConnectionAgeGrace = 5 * time.Millisecond
	maxElapsedTime        = 5 * time.Millisecond
	keepAliveTime         = 5 * time.Millisecond
	keepAliveTimeout      = 1 * time.Millisecond
	connectionType        = "tcp"
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
	CommandService *CommandService
	FileService    *FileService
	GrpcServer     *grpc.Server
}

func NewMockManagementServer(
	apiAddress string,
	agentConfig *config.Config,
) (*MockManagementServer, error) {
	var err error
	commandService := serveCommandService(apiAddress, agentConfig)

	var fileServer *FileService
	if agentConfig.File != nil && agentConfig.File.Location != "" {
		fileServer, err = NewFileService(agentConfig.File.Location)
		if err != nil {
			slog.Error("Failed to create file server", "error", err)
			return nil, err
		}
	}

	grpcListener, err := net.Listen(connectionType,
		fmt.Sprintf("%s:%d", agentConfig.Command.Server.Host, agentConfig.Command.Server.Port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		return nil, err
	}

	grpcServer := grpc.NewServer(getServerOptions(agentConfig)...)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)

	v1.RegisterCommandServiceServer(grpcServer, commandService)
	v1.RegisterFileServiceServer(grpcServer, fileServer)
	go reportHealth(healthcheck, agentConfig)

	go func() {
		slog.Info("gRPC server running", "address", grpcListener.Addr().String())

		err := grpcServer.Serve(grpcListener)
		if err != nil {
			slog.Error("Failed to serve server", "error", err)
		}
	}()

	return &MockManagementServer{
		CommandService: commandService,
		FileService:    fileServer,
		GrpcServer:     grpcServer,
	}, nil
}

func getServerOptions(agentConfig *config.Config) []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpcvalidator.StreamServerInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepAliveEnforcementPolicy),
		grpc.KeepaliveParams(keepAliveServerParameters),
	}

	if agentConfig.Command.Auth == nil || agentConfig.Command.Auth.Token == "" {
		opts = append(opts, grpc.ChainUnaryInterceptor(grpcvalidator.UnaryServerInterceptor()))
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(grpcvalidator.UnaryServerInterceptor(), ensureValidToken))
	}

	return opts
}

func serveCommandService(apiAddress string, agentConfig *config.Config) *CommandService {
	commandServer := NewCommandService()

	go func() {
		cmdListener, listenerErr := createListener(apiAddress, agentConfig)
		if listenerErr != nil {
			return
		}

		if cmdListener != nil {
			commandServer.StartServer(cmdListener)
		}
	}()

	return commandServer
}

func createListener(apiAddress string, agentConfig *config.Config) (net.Listener, error) {
	var listener net.Listener
	var err error

	if agentConfig.Command.TLS != nil && agentConfig.Command.TLS.Enable {
		cert, keyPairErr := tls.LoadX509KeyPair(agentConfig.Command.TLS.Cert, agentConfig.Command.TLS.Key)
		if keyPairErr != nil {
			slog.Error("Failed to load key and cert pair", "error", keyPairErr)
			return nil, keyPairErr
		}

		listener, err = tls.Listen(connectionType, apiAddress, &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		})
	} else {
		listener, err = net.Listen(connectionType, apiAddress)
	}

	if err != nil {
		slog.Error("Failed to create listener", "error", err)
		return nil, err
	}

	return listener, nil
}

func reportHealth(healthcheck *health.Server, agentConfig *config.Config) {
	var sleep time.Duration
	var serverName string
	if agentConfig.Common == nil {
		sleep = maxElapsedTime
	} else {
		sleep = agentConfig.Common.MaxElapsedTime
	}

	if agentConfig.Command.TLS == nil {
		serverName = "test-server"
	} else {
		serverName = agentConfig.Command.TLS.ServerName
	}

	next := healthgrpc.HealthCheckResponse_SERVING
	for {
		healthcheck.SetServingStatus(serverName, next)

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
