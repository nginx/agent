// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"crypto/tls"
	"errors"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/nginx/agent/v3/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// these will come from the agent config
var serviceConfig = `{
	"loadBalancingPolicy": "round_robin",
	"healthCheckConfig": {
		"serviceName": "nginx-agent"
	}
}`

func GetDialOptions(agentConfig *config.Config) []grpc.DialOption {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor()),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	if agentConfig.Client != nil {
		keepAlive := keepalive.ClientParameters{
			Time:                agentConfig.Client.Time, // add to config in future
			Timeout:             agentConfig.Client.Timeout,
			PermitWithoutStream: agentConfig.Client.PermitStream,
		}

		opts = append(opts,
			grpc.WithKeepaliveParams(keepAlive),
		)
	}

	transportCreds, err := getTransportCredentials(agentConfig)
	if err != nil {
		opts = append(opts,
			grpc.WithTransportCredentials(transportCreds),
		)
	}

	if agentConfig.Command.Auth != nil {
		opts = append(opts,
			grpc.WithPerRPCCredentials(
				&PerRPCCredentials{
					Token: agentConfig.Command.Auth.Token,
				}),
		)
	}

	return opts
}

// nolint: ireturn
func getTransportCredentials(agentConfig *config.Config) (credentials.TransportCredentials, error) {
	if agentConfig.Command.TLS == nil {
		return insecure.NewCredentials(), nil
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         agentConfig.Command.Server.Host,
		InsecureSkipVerify: agentConfig.Command.TLS.SkipVerify,
	}

	if agentConfig.Command.TLS.Key == "" {
		return credentials.NewTLS(tlsConfig), nil
	}

	err := appendRootCAs(tlsConfig, agentConfig.Command.TLS.Ca)
	if err != nil {
		return nil, errors.New("unable to append root CA")
	}
	err = appendCertKeyPair(tlsConfig, agentConfig.Command.TLS.Cert, agentConfig.Command.TLS.Key)
	if err != nil {
		return nil, errors.New("unable to append cert and key pair")
	}

	return credentials.NewTLS(tlsConfig), nil
}
