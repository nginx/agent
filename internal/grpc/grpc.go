// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"crypto/tls"
	"errors"
	"log/slog"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpcValidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
	opts := []grpc.DialOption{
		grpc.WithReturnConnectionError(),
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcValidator.UnaryClientInterceptor()),
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

// nolint: ireturn
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
