// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

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
	var opts []grpc.DialOption
	keepAlive := keepalive.ClientParameters{
		Time:                agentConfig.Client.Time, // add to config in future
		Timeout:             agentConfig.Client.Timeout,
		PermitWithoutStream: agentConfig.Client.PermitStream,
	}

	secureDialOption, err := getSecureDialOptions(agentConfig)
	if err == nil {
		opts = append(opts, secureDialOption)
	}

	opts = append(opts,
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor()),
		grpc.WithKeepaliveParams(keepAlive),
		grpc.WithDefaultServiceConfig(serviceConfig),
	)

	return opts
}

func getSecureDialOptions(agentConfig *config.Config) (grpc.DialOption, error) {
	if agentConfig.Command.TLS != nil {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}
	transportCredentials, err := getTransportCredentials(agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to configure tls: %w", err)
	}
	return grpc.WithTransportCredentials(transportCredentials), nil
}

func getTransportCredentials(agentConfig *config.Config) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		// note: ServerName is ignored if InsecureSkipVerify is true
		ServerName:         agentConfig.Command.Server.Host,
		InsecureSkipVerify: agentConfig.Command.TLS.SkipVerify,
	}

	err := appendRootCAs(tlsConfig, agentConfig.Command.TLS.Ca)
	if err != nil {
		return nil, err
	}

	err = appendCertKeyPair(tlsConfig, agentConfig.Command.TLS.Cert, agentConfig.Command.TLS.Key)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlsConfig), nil
}

// appendRootCAs will read, parse, and append any certificates found in the
// file at caFile to the RootCAs in the provided tls Config. If no filepath
// is provided the tls Config is unmodified. By default, if there are no RootCAs
// in the tls Config, it will automatically use the host OS's CA pool.
func appendRootCAs(tlsConfig *tls.Config, caFile string) error {
	if caFile == "" {
		return nil
	}

	ca, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("could not read CA file (%s): %w", caFile, err)
	}

	// If CAs have already been set, append to existing
	caPool := tlsConfig.RootCAs
	if caPool == nil {
		caPool = x509.NewCertPool()
	}

	if !caPool.AppendCertsFromPEM(ca) {
		return fmt.Errorf("could not parse CA cert (%s)", caFile)
	}

	tlsConfig.RootCAs = caPool
	return nil
}

// appendCertKeyPair will attempt to load a cert and key pair from the provided
// file paths and append to the Certificates list in the provided tls Config. If
// no files are provided the tls Config is unmodified. If only one file (key or
// cert) is provided, an error is produced.
func appendCertKeyPair(tlsConfig *tls.Config, certFile, keyFile string) error {
	if certFile == "" && keyFile == "" {
		return nil
	}
	if certFile == "" || keyFile == "" {
		return fmt.Errorf("cert and key must both be provided")
	}

	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("could not load X509 keypair: %w", err)
	}

	tlsConfig.Certificates = append(tlsConfig.Certificates, certificate)
	return nil
}
