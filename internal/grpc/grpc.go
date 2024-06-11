// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/cenkalti/backoff/v4"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . GrpcConnectionInterface

type (
	GrpcConnectionInterface interface {
		CommandServiceClient() mpi.CommandServiceClient
		FileServiceClient() mpi.FileServiceClient
		Close(ctx context.Context) error
	}

	GrpcConnection struct {
		config *config.Config
		conn   *grpc.ClientConn
		mutex  sync.Mutex
	}

	wrappedStream struct {
		grpc.ClientStream
		*protovalidate.Validator
	}
)

var (
	serviceConfig = `{
		"healthCheckConfig": {
			"serviceName": "nginx-agent"
		}
	}`

	defaultCredentials = insecure.NewCredentials()

	_ GrpcConnectionInterface = (*GrpcConnection)(nil)
)

func NewGrpcConnection(ctx context.Context, agentConfig *config.Config) (*GrpcConnection, error) {
	if agentConfig == nil || agentConfig.Command.Server.Type != config.Grpc {
		return nil, errors.New("invalid command server settings")
	}

	grpcConnection := &GrpcConnection{
		config: agentConfig,
	}

	serverAddr := net.JoinHostPort(
		agentConfig.Command.Server.Host,
		fmt.Sprint(agentConfig.Command.Server.Port),
	)

	slog.InfoContext(ctx, "Dialing grpc server", "server_addr", serverAddr)

	info := host.NewInfo()
	resourceID := info.ResourceID(ctx)

	var err error
	grpcConnection.mutex.Lock()
	grpcConnection.conn, err = grpc.NewClient(serverAddr, GetDialOptions(agentConfig, resourceID)...)
	grpcConnection.mutex.Unlock()
	if err != nil {
		return nil, err
	}

	return grpcConnection, nil
}

// nolint: ireturn
func (gc *GrpcConnection) CommandServiceClient() mpi.CommandServiceClient {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	return mpi.NewCommandServiceClient(gc.conn)
}

// nolint: ireturn
func (gc *GrpcConnection) FileServiceClient() mpi.FileServiceClient {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	return mpi.NewFileServiceClient(gc.conn)
}

func (gc *GrpcConnection) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing grpc connection")

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if gc.conn != nil {
		err := gc.conn.Close()
		gc.conn = nil
		if err != nil {
			return fmt.Errorf("gracefully closing gRPC connection: %w", err)
		}
	}

	return nil
}

func (w *wrappedStream) RecvMsg(message any) error {
	err := w.ClientStream.RecvMsg(message)
	if err == nil {
		messageErr := validateMessage(w.Validator, message)
		if messageErr != nil {
			return status.Errorf(
				codes.InvalidArgument,
				"invalid message received from stream: %s",
				messageErr.Error(),
			)
		}
	}

	return err
}

func (w *wrappedStream) SendMsg(message any) error {
	messageErr := validateMessage(w.Validator, message)
	if messageErr != nil {
		return status.Errorf(
			codes.InvalidArgument,
			"invalid message attempted to be sent on stream: %s",
			messageErr.Error(),
		)
	}

	return w.ClientStream.SendMsg(message)
}

func GetDialOptions(agentConfig *config.Config, resourceID string) []grpc.DialOption {
	skipToken := false
	streamClientInterceptors := []grpc.StreamClientInterceptor{grpcRetry.StreamClientInterceptor()}
	unaryClientInterceptors := []grpc.UnaryClientInterceptor{grpcRetry.UnaryClientInterceptor()}

	protoValidatorStreamClientInterceptor, err := ProtoValidatorStreamClientInterceptor()
	if err != nil {
		slog.Error("Unable to add proto validation stream interceptor", "error", err)
	} else {
		streamClientInterceptors = append(streamClientInterceptors, protoValidatorStreamClientInterceptor)
	}

	protoValidatorUnaryClientInterceptor, err := ProtoValidatorUnaryClientInterceptor()
	if err != nil {
		slog.Error("Unable to add proto validation unary interceptor", "error", err)
	} else {
		unaryClientInterceptors = append(unaryClientInterceptors, protoValidatorUnaryClientInterceptor)
	}

	opts := []grpc.DialOption{
		grpc.WithChainStreamInterceptor(streamClientInterceptors...),
		grpc.WithChainUnaryInterceptor(unaryClientInterceptors...),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	if agentConfig.Client != nil {
		keepAlive := keepalive.ClientParameters{
			Time:                agentConfig.Client.Time,
			Timeout:             agentConfig.Client.Timeout,
			PermitWithoutStream: agentConfig.Client.PermitWithoutStream,
		}

		opts = append(opts,
			grpc.WithKeepaliveParams(keepAlive),
		)
	}

	transportCredentials, err := getTransportCredentials(agentConfig)
	if err == nil {
		slog.Debug("Adding transport credentials to gRPC dial options")
		opts = append(opts,
			grpc.WithTransportCredentials(transportCredentials),
		)
	} else {
		slog.Debug("Adding default transport credentials to gRPC dial options")
		opts = append(opts,
			grpc.WithTransportCredentials(defaultCredentials),
		)
		skipToken = true
	}

	if agentConfig.Command.Auth != nil && !skipToken {
		slog.Debug("Adding token to RPC credentials")
		opts = append(opts,
			grpc.WithPerRPCCredentials(
				&PerRPCCredentials{
					Token: agentConfig.Command.Auth.Token,
					ID:    resourceID,
				}),
		)
	}

	return opts
}

// Have to create our own UnaryClientInterceptor function since protovalidate only provides a UnaryServerInterceptor
// https://pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/v2@v2.1.0/interceptors/protovalidate
func ProtoValidatorUnaryClientInterceptor() (grpc.UnaryClientInterceptor, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		requestValidationErr := validateMessage(validator, req)
		if requestValidationErr != nil {
			return status.Errorf(
				codes.InvalidArgument,
				"invalid request message: %s",
				requestValidationErr.Error(),
			)
		}

		invokerErr := invoker(ctx, method, req, reply, cc, opts...)
		if invokerErr != nil {
			return invokerErr
		}

		replyValidationErr := validateMessage(validator, reply)
		if replyValidationErr != nil {
			return status.Errorf(
				codes.InvalidArgument,
				"invalid reply message: %s",
				replyValidationErr.Error(),
			)
		}

		return nil
	}, nil
}

func ProtoValidatorStreamClientInterceptor() (grpc.StreamClientInterceptor, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		clientStream, streamerError := streamer(ctx, desc, cc, method, opts...)
		if streamerError != nil {
			return nil, streamerError
		}

		return &wrappedStream{clientStream, validator}, nil
	}, nil
}

func ValidateGrpcError(err error) error {
	if err != nil {
		if statusError, ok := status.FromError(err); ok {
			if statusError.Code() == codes.InvalidArgument || statusError.Code() == codes.Unimplemented {
				return backoff.Permanent(err)
			}
		}

		return err
	}

	return nil
}

func validateMessage(validator *protovalidate.Validator, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "invalid request type: %T", message)
	}

	validationErr := validator.Validate(protoMessage)
	if validationErr != nil {
		return status.Errorf(codes.InvalidArgument, validationErr.Error())
	}

	return nil
}

func getTransportCredentials(agentConfig *config.Config) (credentials.TransportCredentials, error) {
	if agentConfig.Command.TLS == nil {
		return defaultCredentials, nil
	}

	if agentConfig.Command.TLS.SkipVerify {
		slog.Warn("Verification of the server's certificate chain and host name is disabled")
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         agentConfig.Command.TLS.ServerName,
		InsecureSkipVerify: agentConfig.Command.TLS.SkipVerify,
	}

	if agentConfig.Command.TLS.Key == "" {
		return credentials.NewTLS(tlsConfig), nil
	}

	err := appendCertKeyPair(tlsConfig, agentConfig.Command.TLS.Cert, agentConfig.Command.TLS.Key)
	if err != nil {
		return nil, errors.New("append cert and key pair")
	}

	err = appendRootCAs(tlsConfig, agentConfig.Command.TLS.Ca)
	if err != nil {
		slog.Debug("Unable to append root CA", "error", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}
