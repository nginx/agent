// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	testUUID   = "uuid"
	testToken  = "token"
	testBearer = "bearer"
	testMethod = "/some/method"
)

func mockInvoker(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	return nil
}

func mockStreamer(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func TestClientInterceptor_Unary(t *testing.T) {
	clientInterceptor := NewClientAuth(testUUID, testToken)
	err := clientInterceptor.Unary()(context.Background(), testMethod, struct{}{}, struct{}{}, &grpc.ClientConn{}, mockInvoker, []grpc.CallOption{}...)
	require.NoError(t, err)
}

func TestClientInterceptor_Stream(t *testing.T) {
	clientInterceptor := NewClientAuth(testUUID, testToken)
	interceptor, err := clientInterceptor.Stream()(context.Background(), &grpc.StreamDesc{}, &grpc.ClientConn{}, testMethod, mockStreamer, []grpc.CallOption{}...)
	require.NoError(t, err)

	// assert that the mockStreamer function was called
	assert.Nil(t, interceptor)
}

func TestClientInterceptor_GetRequestMetadata(t *testing.T) {
	clientInterceptor := NewClientAuth(testUUID, testToken, WithBearerToken(testBearer))

	metadataResult, err := clientInterceptor.GetRequestMetadata(context.Background())
	require.NoError(t, err)

	expectedMetadata := map[string]string{
		TokenHeader:  testToken,
		IDHeader:     testUUID,
		BearerHeader: testBearer,
	}

	for key, expectedValue := range expectedMetadata {
		assert.Equal(t, expectedValue, metadataResult[key])
	}
}

func TestClientInterceptor_RequireTransportSecurity(t *testing.T) {
	interceptor := NewClientAuth(testUUID, testToken)
	requireTLS := interceptor.RequireTransportSecurity()
	assert.False(t, requireTLS)
}
