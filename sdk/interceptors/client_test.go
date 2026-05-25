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

func TestNewClientAuth_Variants(t *testing.T) {
	tests := []struct {
		name         string
		options      []Option
		wantUUID     string
		wantToken    string
		wantBearer   string
		checkLogger  bool
		customLogger *log.Logger
	}{
		{
			name:         "DefaultLogger",
			options:      nil,
			wantUUID:     testUUID,
			wantToken:    testToken,
			wantBearer:   "",
			checkLogger:  true,
			customLogger: nil,
		},
		{
			name:         "WithBearerToken",
			options:      []Option{WithBearerToken(testBearer)},
			wantUUID:     testUUID,
			wantToken:    testToken,
			wantBearer:   testBearer,
			checkLogger:  false,
			customLogger: nil,
		},
		{
			name:         "MultipleOptions_LastWins",
			options:      []Option{WithBearerToken("first"), WithBearerToken("second")},
			wantUUID:     testUUID,
			wantToken:    testToken,
			wantBearer:   "second",
			checkLogger:  false,
			customLogger: nil,
		},
		{
			name:         "CustomLogger",
			options:      []Option{withLoggerForTest(log.New())},
			wantUUID:     testUUID,
			wantToken:    testToken,
			wantBearer:   "",
			checkLogger:  false,
			customLogger: log.New(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := NewClientAuth(testUUID, testToken, tt.options...)
			require.NotNil(t, ci)
			assert.Equal(t, tt.wantUUID, ci.uuid)
			assert.Equal(t, tt.wantToken, ci.token)
			assert.Equal(t, tt.wantBearer, ci.bearer)
			if tt.checkLogger {
				assert.NotNil(t, ci.log, "default logger should be assigned")
			}
			if tt.customLogger != nil {
				// We can't compare loggers by value, but we can check type and that it's not the default
				assert.IsType(t, tt.customLogger, ci.log)
			}
		})
	}
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

func withLoggerForTest(l *log.Logger) Option {
	return func(opt *option) {
		opt.client.log = l
	}
}
