/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package interceptors

import (
	"context"
	"errors"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	testUUID   = "test-uuid-1234"
	testToken  = "test-token-abcd"
	testBearer = "test-bearer-xyz"
)

func TestNewClientAuth_DefaultLogger(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken)

	require.NotNil(t, ci)
	assert.Equal(t, testUUID, ci.uuid)
	assert.Equal(t, testToken, ci.token)
	assert.Empty(t, ci.bearer)
	assert.NotNil(t, ci.log, "default logger should be assigned")
}

func TestNewClientAuth_WithBearerToken(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken, WithBearerToken(testBearer))

	require.NotNil(t, ci)
	assert.Equal(t, testBearer, ci.bearer)
}

func TestNewClientAuth_MultipleOptions(t *testing.T) {
	// Applying option multiple times: last one wins.
	ci := NewClientAuth(testUUID, testToken,
		WithBearerToken("first"),
		WithBearerToken("second"),
	)

	require.NotNil(t, ci)
	assert.Equal(t, "second", ci.bearer)
}

func TestClientInterceptor_GetRequestMetadata(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken, WithBearerToken(testBearer))

	md, err := ci.GetRequestMetadata(context.Background())
	require.NoError(t, err)

	assert.Equal(t, testToken, md[TokenHeader])
	assert.Equal(t, testUUID, md[IDHeader])
	assert.Equal(t, testBearer, md[BearerHeader])
	assert.Len(t, md, 3)
}

func TestClientInterceptor_GetRequestMetadata_IgnoresURIArgs(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken)

	md, err := ci.GetRequestMetadata(context.Background(), "ignored", "args")
	require.NoError(t, err)
	assert.Equal(t, testToken, md[TokenHeader])
}

func TestClientInterceptor_RequireTransportSecurity_ReturnsFalse(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken)
	assert.False(t, ci.RequireTransportSecurity())
}

func TestClientInterceptor_Unary_AttachesMetadata(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken, WithBearerToken(testBearer))
	unary := ci.Unary()
	require.NotNil(t, unary)

	var capturedCtx context.Context
	invoker := func(ctx context.Context, _ string, _, _ interface{},
		_ *grpc.ClientConn, _ ...grpc.CallOption,
	) error {
		capturedCtx = ctx
		return nil
	}

	err := unary(context.Background(), "/svc/Method", nil, nil, nil, invoker)
	require.NoError(t, err)

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	require.True(t, ok, "outgoing metadata should be present")
	assert.Equal(t, []string{testToken}, md.Get(TokenHeader))
	assert.Equal(t, []string{testUUID}, md.Get(IDHeader))
	assert.Equal(t, []string{testBearer}, md.Get(BearerHeader))
}

func TestClientInterceptor_Unary_PropagatesInvokerError(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken)
	wantErr := errors.New("invoker exploded")

	invoker := func(_ context.Context, _ string, _, _ interface{},
		_ *grpc.ClientConn, _ ...grpc.CallOption,
	) error {
		return wantErr
	}

	err := ci.Unary()(context.Background(), "/svc/Method", nil, nil, nil, invoker)
	assert.ErrorIs(t, err, wantErr)
}

func TestClientInterceptor_Stream_AttachesMetadata(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken, WithBearerToken(testBearer))
	stream := ci.Stream()
	require.NotNil(t, stream)

	var capturedCtx context.Context
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn,
		_ string, _ ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		capturedCtx = ctx
		return nil, nil
	}

	_, err := stream(context.Background(), &grpc.StreamDesc{}, nil, "/svc/Stream", streamer)
	require.NoError(t, err)

	md, ok := metadata.FromOutgoingContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, []string{testToken}, md.Get(TokenHeader))
	assert.Equal(t, []string{testUUID}, md.Get(IDHeader))
	assert.Equal(t, []string{testBearer}, md.Get(BearerHeader))
}

func TestClientInterceptor_AttachToken_PreservesExistingMetadata(t *testing.T) {
	ci := NewClientAuth(testUUID, testToken)

	existing := metadata.Pairs("custom-key", "custom-value")
	ctx := metadata.NewOutgoingContext(context.Background(), existing)

	out := ci.attachToken(ctx)
	md, ok := metadata.FromOutgoingContext(out)
	require.True(t, ok)

	assert.Equal(t, []string{"custom-value"}, md.Get("custom-key"))
	assert.Equal(t, []string{testToken}, md.Get(TokenHeader))
}

func TestClientInterceptor_CustomLogger(t *testing.T) {
	customLogger := log.New()
	ci := NewClientAuth(testUUID, testToken, withLoggerForTest(customLogger))

	assert.Same(t, customLogger, ci.log)
}

func withLoggerForTest(l *log.Logger) Option {
	return func(opt *option) {
		opt.client.log = l
	}
}
