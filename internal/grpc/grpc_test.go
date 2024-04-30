// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ClientStream struct{}

func (*ClientStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (*ClientStream) Trailer() metadata.MD {
	return nil
}

func (*ClientStream) CloseSend() error {
	return nil
}

func (*ClientStream) Context() context.Context {
	return nil
}

func (*ClientStream) SendMsg(m any) error {
	return nil
}

func (*ClientStream) RecvMsg(m any) error {
	return nil
}

func Test_GetDialOptions(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		expected    int
		createCerts bool
	}{
		{
			"Test 1: DialOptions insecure",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: types.GetAgentConfig().Command.Server,
					Auth:   types.GetAgentConfig().Command.Auth,
					TLS: &config.TLSConfig{
						Cert:       "some.cert",
						Key:        "some.key",
						Ca:         "some.ca",
						SkipVerify: false,
						ServerName: "server-name",
					},
				},
			},
			6,
			false,
		},
		{
			"Test 2: DialOptions mTLS",
			types.GetAgentConfig(),
			6,
			true,
		},
		{
			"Test 3: DialOptions TLS",
			&config.Config{
				Command: &config.Command{
					Server: types.GetAgentConfig().Command.Server,
					Auth:   types.GetAgentConfig().Command.Auth,
					TLS: &config.TLSConfig{
						Cert:       "some.cert",
						Key:        "some.key",
						Ca:         "some.ca",
						SkipVerify: false,
						ServerName: "server-name",
					},
				},
				Client: types.GetAgentConfig().Client,
			},
			6,
			false,
		},
		{
			"Test 4: DialOptions No Client",
			&config.Config{
				Command: &config.Command{
					Server: types.GetAgentConfig().Command.Server,
					Auth:   types.GetAgentConfig().Command.Auth,
					TLS:    types.GetAgentConfig().Command.TLS,
				},
			},
			5,
			false,
		},
		{
			"Test 5: DialOptions No Auth",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: types.GetAgentConfig().Command.Server,
					TLS:    types.GetAgentConfig().Command.TLS,
				},
			},
			6,
			false,
		},
		{
			"Test 6: DialOptions No TLS",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: types.GetAgentConfig().Command.Server,
					Auth:   types.GetAgentConfig().Command.Auth,
				},
			},
			7,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			if test.createCerts {
				tmpDir := t.TempDir()
				// not mTLS scripts
				key, cert := helpers.GenerateSelfSignedCert(t)
				_, ca := helpers.GenerateSelfSignedCert(t)

				keyContents := helpers.Cert{Name: keyFileName, Type: certificateType, Contents: key}
				certContents := helpers.Cert{Name: certFileName, Type: privateKeyType, Contents: cert}
				caContents := helpers.Cert{Name: caFileName, Type: certificateType, Contents: ca}

				helpers.WriteCertFiles(t, tmpDir, keyContents)
				helpers.WriteCertFiles(t, tmpDir, certContents)
				helpers.WriteCertFiles(t, tmpDir, caContents)

				test.agentConfig.Command.TLS.Cert = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, certFileName)
				test.agentConfig.Command.TLS.Key = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, keyFileName)
				test.agentConfig.Command.TLS.Ca = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, caFileName)
			}

			options := GetDialOptions(test.agentConfig)
			assert.NotNil(tt, options)
			assert.Len(tt, options, test.expected)
		})
	}
}

func Test_ProtoValidatorUnaryClientInterceptor(t *testing.T) {
	ctx := context.Background()
	interceptor, err := ProtoValidatorUnaryClientInterceptor()
	require.NoError(t, err)

	invoker := func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return nil
	}

	tests := []struct {
		name            string
		request         any
		reply           any
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid request type",
			request:         "invalid",
			reply:           protos.GetNginxOssInstance([]string{}),
			isErrorExpected: true,
		},
		{
			name:            "Test 2: Invalid reply type",
			request:         protos.GetNginxOssInstance([]string{}),
			reply:           "invalid",
			isErrorExpected: true,
		},
		{
			name:            "Test 3: Valid request & reply types",
			request:         protos.GetNginxOssInstance([]string{}),
			reply:           protos.GetNginxOssInstance([]string{}),
			isErrorExpected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			validationError := interceptor(ctx, "", test.request, test.reply, nil, invoker, nil)
			validateError(tt, validationError, test.isErrorExpected)
		})
	}
}

func Test_ProtoValidatorStreamClientInterceptor_RecvMsg(t *testing.T) {
	ctx := context.Background()
	interceptor, err := ProtoValidatorStreamClientInterceptor()
	require.NoError(t, err)

	tests := []struct {
		name            string
		receivedMessage any
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid received message type",
			receivedMessage: "invalid",
			isErrorExpected: true,
		}, {
			name:            "Test 2: Valid received message type",
			receivedMessage: protos.GetNginxOssInstance(),
			isErrorExpected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			clientStream := createStreamInterceptor(tt, ctx, interceptor)

			validationError := clientStream.RecvMsg(test.receivedMessage)
			validateError(tt, validationError, test.isErrorExpected)
		})
	}
}

func Test_ProtoValidatorStreamClientInterceptor_SendMsg(t *testing.T) {
	ctx := context.Background()
	interceptor, err := ProtoValidatorStreamClientInterceptor()
	require.NoError(t, err)

	tests := []struct {
		name            string
		sentMessage     any
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid sent message type",
			sentMessage:     "invalid",
			isErrorExpected: true,
		}, {
			name:            "Test 2: Valid sent message type",
			sentMessage:     protos.GetNginxOssInstance(),
			isErrorExpected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			clientStream := createStreamInterceptor(tt, ctx, interceptor)

			validationError := clientStream.SendMsg(test.sentMessage)
			validateError(tt, validationError, test.isErrorExpected)
		})
	}
}

func createStreamInterceptor(
	t *testing.T,
	ctx context.Context,
	interceptor grpc.StreamClientInterceptor,
) grpc.ClientStream {
	t.Helper()

	streamer := func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return &ClientStream{}, nil
	}

	clientStream, interceptorError := interceptor(
		ctx,
		&grpc.StreamDesc{},
		&grpc.ClientConn{},
		"",
		streamer,
		[]grpc.CallOption{}...,
	)
	require.NoError(t, interceptorError)
	assert.NotNil(t, clientStream)

	return clientStream
}

func validateError(t *testing.T, validationError error, isErrorExpected bool) {
	t.Helper()

	t.Log(validationError)

	assert.Equal(t, isErrorExpected, validationError != nil)

	if validationError != nil {
		if err, ok := status.FromError(validationError); ok {
			assert.Equal(t, codes.InvalidArgument, err.Code())
		}
	}
}
