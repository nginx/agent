/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math"
	"os"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/nginx/agent/sdk/v2/proto"
)

type clientAuth struct {
	UUID  string
	Token string
}

var (
	// DefaultClientDialOptions are default settings for a connection to the dataplane
	DefaultClientDialOptions = []grpc.DialOption{
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                120 * time.Second,
			Timeout:             60 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
	}

	DefaultServerDialOptions = []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             60 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxSendMsgSize(math.MaxInt),
		grpc.MaxRecvMsgSize(math.MaxInt),
	}
)

// NewGrpcConnection -
func NewGrpcConnection(target string, dialOptions []grpc.DialOption) (*grpc.ClientConn, error) {
	if dialOptions == nil {
		dialOptions = DefaultClientDialOptions
	}

	return NewGrpcConnectionWithContext(context.TODO(), target, dialOptions)
}

// NewGrpcConnectionWithContext -
func NewGrpcConnectionWithContext(ctx context.Context, server string, dialOptions []grpc.DialOption) (*grpc.ClientConn, error) {
	if dialOptions == nil {
		dialOptions = DefaultClientDialOptions
	}

	return grpc.DialContext(ctx, server, dialOptions...)
}

// SecureDialOptions returns dialOptions with tls support
func SecureDialOptions(tlsEnabled bool, certPath string, keyPath string, caPath string, serverName string, skipVerify bool) (grpc.DialOption, error) {
	if !tlsEnabled {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}
	transCreds, err := getTransportCredentials(certPath, keyPath, caPath, serverName, skipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to configure tls: %w", err)
	}
	return grpc.WithTransportCredentials(transCreds), nil
}

// DataplaneConnectionDialOptions returns dialOptions for connecting to a dataplane instance
func DataplaneConnectionDialOptions(Token string, meta *proto.Metadata) []grpc.DialOption {
	dataplaneDialOptions := []grpc.DialOption{}
	if Token != "" {
		c := &clientAuth{UUID: meta.GetClientId(), Token: Token}

		authInterceptor := interceptors.NewClientAuth(c.UUID, c.Token, []interceptors.Option{
			interceptors.WithBearerToken(c.Token),
		}...)

		dataplaneDialOptions = []grpc.DialOption{
			grpc.WithStreamInterceptor(authInterceptor.Stream()),
			grpc.WithUnaryInterceptor(authInterceptor.Unary()),
		}
	}
	return dataplaneDialOptions
}

// GetCallOptions -
func GetCallOptions() []grpc.CallOption {
	callOptions := []grpc.CallOption{
		grpc_retry.WithCodes(codes.NotFound),
		grpc.WaitForReady(true),
	}
	return callOptions
}

// GetCommandClient returns a commanderClient with that grpc connection
func GetCommandClient(conn *grpc.ClientConn) proto.CommanderClient {
	return proto.NewCommanderClient(conn)
}

// GetCommandChannel returns a channel that commands are sent over
func GetCommandChannel(client proto.CommanderClient) (proto.Commander_CommandChannelClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return client.CommandChannel(ctx, GetCallOptions()...)
}

func getTransportCredentials(certPath string, keyPath string, caPath string, serverName string, skipVerify bool) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		// note: ServerName is ignored if InsecureSkipVerify is true
		ServerName:         serverName,
		InsecureSkipVerify: skipVerify,
	}

	err := appendRootCAs(tlsConfig, caPath)
	if err != nil {
		return nil, err
	}

	err = appendCertKeyPair(tlsConfig, certPath, keyPath)
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
// filepaths and append to the Certificates list in the provided tls Config. If
// no files are provided the tls Config is unmodified. If only one file (key or
// cert) is provided, an error is produced.
func appendCertKeyPair(tlsConfig *tls.Config, certFile string, keyFile string) error {
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
