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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nginx/agent/v3/internal/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/bufbuild/protovalidate-go"
	protovalidateInterceptor "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	grpcvalidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	maxElapsedTime   = 5 * time.Second
	keepAliveTime    = 5 * time.Second
	keepAliveTimeout = 10 * time.Second
	testTimeout      = 100 * time.Millisecond
	connectionType   = "tcp"
)

var keepAliveEnforcementPolicy = keepalive.EnforcementPolicy{
	MinTime:             keepAliveTime,
	PermitWithoutStream: true,
}

var keepAliveServerParameters = keepalive.ServerParameters{
	Time:    keepAliveTime,
	Timeout: keepAliveTimeout,
}

var (
	commandServiceLock sync.Mutex
	fileServiceLock    sync.Mutex
)

type MockManagementServer struct {
	CommandService *CommandService
	FileService    *FileService
	GrpcServer     *grpc.Server
}

func NewMockManagementServer(
	apiAddress string,
	agentConfig *config.Config,
	configDirectory *string,
) (*MockManagementServer, error) {
	var err error
	commandService := serveCommandService(apiAddress, agentConfig)

	var fileServer *FileService

	if *configDirectory != "" {
		fileServer, err = NewFileService(*configDirectory)
		if err != nil {
			slog.Error("Failed to create file server", "error", err)
			return nil, err
		}
	}

	fileServiceLock.Lock()
	defer fileServiceLock.Unlock()

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
		slog.Info("Starting mock management plane gRPC server", "address", grpcListener.Addr().String())
		grpcErr := grpcServer.Serve(grpcListener)
		if grpcErr != nil {
			slog.Error("Failed to start mock management plane gRPC server", "error", grpcErr)
		}
	}()

	return &MockManagementServer{
		CommandService: commandService,
		FileService:    fileServer,
		GrpcServer:     grpcServer,
	}, nil
}

func (ms *MockManagementServer) Stop() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		signal.Stop(sigs)

		ms.GrpcServer.Stop()

		time.Sleep(testTimeout)
		done <- true
	}()

	<-done
	ms.GrpcServer.GracefulStop()

	time.Sleep(testTimeout)
}

func getServerOptions(agentConfig *config.Config) []grpc.ServerOption {
	validator, _ := protovalidate.New()

	opts := []grpc.ServerOption{
		grpc.ChainStreamInterceptor(
			grpcvalidator.StreamServerInterceptor(),
			protovalidateInterceptor.StreamServerInterceptor(validator),
		),
		grpc.KeepaliveEnforcementPolicy(keepAliveEnforcementPolicy),
		grpc.KeepaliveParams(keepAliveServerParameters),
	}

	if agentConfig.Command.Auth == nil || agentConfig.Command.Auth.Token == "" {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			grpcvalidator.UnaryServerInterceptor(),
			protovalidateInterceptor.UnaryServerInterceptor(validator),
		),
		)
	} else {
		opts = append(opts, grpc.ChainUnaryInterceptor(
			grpcvalidator.UnaryServerInterceptor(),
			protovalidateInterceptor.UnaryServerInterceptor(validator),
			ensureValidToken,
		),
		)
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
			commandServiceLock.Lock()
			defer commandServiceLock.Unlock()
			commandServer.StartServer(cmdListener)
		}
	}()

	return commandServer
}

func createListener(apiAddress string, agentConfig *config.Config) (net.Listener, error) {
	if agentConfig.Command.TLS != nil {
		cert, keyPairErr := tls.LoadX509KeyPair(agentConfig.Command.TLS.Cert, agentConfig.Command.TLS.Key)

		if keyPairErr == nil {
			slog.Error("Failed to load key and cert pair", "error", keyPairErr)
			return tls.Listen(connectionType, apiAddress, &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			})
		}
	}

	return net.Listen(connectionType, apiAddress)
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
