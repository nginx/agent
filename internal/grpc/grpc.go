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

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// these will come from the agent config
var serviceConfig = `{
	"healthCheckConfig": {
		"serviceName": "nginx-agent"
	}
}`
var defaultCredentials = insecure.NewCredentials()

func GetDialOptions(agentConfig *config.Config) []grpc.DialOption {
	skipToken := false
	unaryClientInterceptors := []grpc.UnaryClientInterceptor{grpcRetry.UnaryClientInterceptor()}

	protoValidatorUnaryClientInterceptor, err := ProtoValidatorUnaryClientInterceptor()
	if err != nil {
		slog.Error("Unable to add proto validation interceptor", "error", err)
	} else {
		unaryClientInterceptors = append(unaryClientInterceptors, protoValidatorUnaryClientInterceptor)
	}

	opts := []grpc.DialOption{
		grpc.WithReturnConnectionError(),
		grpc.WithChainStreamInterceptor(grpcRetry.StreamClientInterceptor()),
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
					ID:    agentConfig.UUID,
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
		slog.Debug("Validation interceptor request", "req", req)

		requestValidationErr := validateMessage(validator, req)
		if requestValidationErr != nil {
			return status.Errorf(
				codes.InvalidArgument,
				fmt.Errorf("invalid request message: %w", requestValidationErr).Error(),
			)
		}

		invokerErr := invoker(ctx, method, req, reply, cc, opts...)
		if invokerErr != nil {
			return invokerErr
		}

		slog.Debug("Validation interceptor reply", "reply", reply)

		replyValidationErr := validateMessage(validator, reply)
		if replyValidationErr != nil {
			return status.Errorf(
				codes.InvalidArgument,
				fmt.Errorf("invalid reply message: %w", replyValidationErr).Error(),
			)
		}

		return nil
	}, nil
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
