/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package interceptors

import (
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// TokenHeader in outbound metadata for an authorization token
	TokenHeader = "authorization"
	// IDHeader in an outbound metadata for a client ID
	IDHeader = "uuid"
	// BearerHeader in an outbound metadata for a bearer token (typically a JWT)
	BearerHeader = "bearer"
)

type clientInterceptor struct {
	log    *log.Logger
	uuid   string
	token  string
	bearer string
}

// NewClientAuth for outbound authenticated connections
func NewClientAuth(uuid, token string, opts ...Option) *clientInterceptor {
	opt := &option{client: &clientInterceptor{uuid: uuid, token: token}}
	for _, o := range opts {
		o(opt)
	}
	if opt.client.log == nil {
		opt.client.log = log.New()
	}
	return opt.client
}

// WithBearerToken to skip the API Token auth
func WithBearerToken(bearer string) Option {
	return func(opt *option) {
		opt.client.bearer = bearer
	}
}

func (c *clientInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		c.log.Debugf("--> client unary interceptor: %s", method)
		return invoker(c.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

func (c *clientInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		c.log.Debugf("--> client stream interceptor: %s", method)

		return streamer(c.attachToken(ctx), desc, cc, method, opts...)
	}
}

// GetRequestMetadata satisfy the interface grpc.PerRPCCredentials, by setting the auth token, and client id for the
// context. see: https://godoc.org/google.golang.org/grpc/credentials#PerRPCCredentials
func (c *clientInterceptor) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		TokenHeader:  c.token,
		IDHeader:     c.uuid,
		BearerHeader: c.bearer,
	}, nil
}

// RequireTransportSecurity satisfy the interface grpc.PerRPCCredentials.
func (c *clientInterceptor) RequireTransportSecurity() bool {
	return false
}

func (c *clientInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		TokenHeader, c.token,
		IDHeader, c.uuid,
		BearerHeader, c.bearer,
	)
}

type option struct {
	client *clientInterceptor
}

// Authenticator Auth-s the initial connection then allows validation at any point of the stream
type Authenticator interface {
	Auth(ctx context.Context) error
	ValidateClientToken(ctx context.Context) (Claims, error)
}

// Claims contain information about the VerifiedClientToken
type Claims map[string]interface{}

type Option func(opt *option)
