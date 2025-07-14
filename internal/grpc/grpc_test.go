// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ClientStream struct{}

func (*ClientStream) Header() (metadata.MD, error) {
	// nolint: nilnil
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

type TestError struct{}

func (z TestError) Error() string {
	return "Test"
}

func Test_GrpcConnection(t *testing.T) {
	ctx := context.Background()

	conn, err := NewGrpcConnection(ctx, types.AgentConfig(), types.AgentConfig().Command)

	require.NoError(t, err)
	assert.NotNil(t, conn)

	assert.NotNil(t, conn.CommandServiceClient())
	assert.NotNil(t, conn.FileServiceClient())

	require.NoError(t, conn.Close(ctx))
}

func Test_GetDialOptions(t *testing.T) {
	tests := []struct {
		agentConfig *config.Config
		name        string
		expected    int
		createCerts bool
	}{
		{
			name: "Test 1: DialOptions insecure",
			agentConfig: &config.Config{
				Client: types.AgentConfig().Client,
				Command: &config.Command{
					Server: types.AgentConfig().Command.Server,
					Auth:   types.AgentConfig().Command.Auth,
					TLS: &config.TLSConfig{
						Cert:       "some.cert",
						Key:        "some.key",
						Ca:         "some.ca",
						SkipVerify: false,
						ServerName: "server-name",
					},
				},
			},
			expected:    7,
			createCerts: false,
		},
		{
			name:        "Test 2: DialOptions mTLS",
			agentConfig: types.AgentConfig(),
			expected:    8,
			createCerts: true,
		},
		{
			name: "Test 3: DialOptions TLS",
			agentConfig: &config.Config{
				Command: &config.Command{
					Server: types.AgentConfig().Command.Server,
					Auth:   types.AgentConfig().Command.Auth,
					TLS: &config.TLSConfig{
						Cert:       "some.cert",
						Key:        "some.key",
						Ca:         "some.ca",
						SkipVerify: false,
						ServerName: "server-name",
					},
				},
				Client: types.AgentConfig().Client,
			},
			expected:    7,
			createCerts: false,
		},
		{
			name: "Test 4: DialOptions No Client",
			agentConfig: &config.Config{
				Command: &config.Command{
					Server: types.AgentConfig().Command.Server,
					Auth:   types.AgentConfig().Command.Auth,
					TLS:    types.AgentConfig().Command.TLS,
				},
			},
			expected:    5,
			createCerts: false,
		},
		{
			name: "Test 5: DialOptions No Auth",
			agentConfig: &config.Config{
				Client: types.AgentConfig().Client,
				Command: &config.Command{
					Server: types.AgentConfig().Command.Server,
					TLS:    types.AgentConfig().Command.TLS,
				},
			},
			expected:    7,
			createCerts: false,
		},
		{
			name: "Test 6: DialOptions No TLS",
			agentConfig: &config.Config{
				Client: types.AgentConfig().Client,
				Command: &config.Command{
					Server: types.AgentConfig().Command.Server,
					Auth:   types.AgentConfig().Command.Auth,
				},
			},
			expected:    8,
			createCerts: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			if test.createCerts {
				tmpDir := t.TempDir()
				// not mTLS scripts
				key, cert := helpers.GenerateSelfSignedCert(t)
				_, ca := helpers.GenerateSelfSignedCert(t)

				keyContents := helpers.Cert{Name: keyFileName, Type: privateKeyType, Contents: key}
				certContents := helpers.Cert{Name: certFileName, Type: certificateType, Contents: cert}
				caContents := helpers.Cert{Name: caFileName, Type: certificateType, Contents: ca}

				helpers.WriteCertFiles(t, tmpDir, keyContents)
				helpers.WriteCertFiles(t, tmpDir, certContents)
				helpers.WriteCertFiles(t, tmpDir, caContents)

				test.agentConfig.Command.TLS.Cert = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, certFileName)
				test.agentConfig.Command.TLS.Key = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, keyFileName)
				test.agentConfig.Command.TLS.Ca = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, caFileName)
			}

			options := DialOptions(test.agentConfig, test.agentConfig.Command, "123")
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
		request         any
		reply           any
		name            string
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid request type",
			request:         "invalid",
			reply:           protos.NginxOssInstance([]string{}),
			isErrorExpected: true,
		},
		{
			name:            "Test 2: Invalid reply type",
			request:         protos.NginxOssInstance([]string{}),
			reply:           "invalid",
			isErrorExpected: true,
		},
		{
			name:            "Test 3: Valid request & reply types",
			request:         protos.NginxOssInstance([]string{}),
			reply:           protos.NginxOssInstance([]string{}),
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
		receivedMessage any
		name            string
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid received message type",
			receivedMessage: "invalid",
			isErrorExpected: true,
		}, {
			name:            "Test 2: Valid received message type",
			receivedMessage: protos.NginxOssInstance([]string{}),
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
		sentMessage     any
		name            string
		isErrorExpected bool
	}{
		{
			name:            "Test 1: Invalid sent message type",
			sentMessage:     "invalid",
			isErrorExpected: true,
		}, {
			name:            "Test 2: Valid sent message type",
			sentMessage:     protos.NginxOssInstance([]string{}),
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

func Test_ValidateGrpcError(t *testing.T) {
	result := ValidateGrpcError(TestError{})
	assert.IsType(t, TestError{}, result)

	result = ValidateGrpcError(status.Errorf(codes.InvalidArgument, "error"))
	assert.IsType(t, &backoff.PermanentError{}, result)
}

func Test_transportCredentials(t *testing.T) {
	tests := map[string]struct {
		conf                *config.Config
		wantSecurityProfile string
		wantServerName      string
		wantErr             bool
	}{
		"Test 1: No TLS config returns default credentials": {
			conf: &config.Config{
				Command: &config.Command{},
			},
			wantErr:             false,
			wantSecurityProfile: "insecure",
		},
		"Test 2: With tls config returns secure credentials": {
			conf: &config.Config{
				Command: &config.Command{
					TLS: &config.TLSConfig{
						ServerName: "foobar",
						SkipVerify: true,
					},
				},
			},
			wantErr:             false,
			wantSecurityProfile: "tls",
		},
		"Test 3: With invalid tls config should error": {
			conf:    types.AgentConfig(), // references non-existent certs
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := transportCredentials(tt.conf.Command)
			if tt.wantErr {
				require.Error(t, err, "transportCredentials(%v)", tt.conf)

				return
			}
			require.NoError(t, err, "transportCredentials(%v)", tt.conf)
			require.Equal(t, tt.wantSecurityProfile, got.Info().SecurityProtocol, "incorrect SecurityProtocol")
		})
	}
}

func Test_tlsConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// not mTLS scripts
	key, cert := helpers.GenerateSelfSignedCert(t)
	_, ca := helpers.GenerateSelfSignedCert(t)

	keyContents := helpers.Cert{Name: keyFileName, Type: privateKeyType, Contents: key}
	certContents := helpers.Cert{Name: certFileName, Type: certificateType, Contents: cert}
	caContents := helpers.Cert{Name: caFileName, Type: certificateType, Contents: ca}

	keyPath := helpers.WriteCertFiles(t, tmpDir, keyContents)
	certPath := helpers.WriteCertFiles(t, tmpDir, certContents)
	caPath := helpers.WriteCertFiles(t, tmpDir, caContents)

	tests := map[string]struct {
		conf    *config.TLSConfig
		verify  func(require.TestingT, *tls.Config)
		wantErr bool
	}{
		"Test 1: all config should be translated": {
			conf: &config.TLSConfig{
				Cert:       certPath,
				Key:        keyPath,
				Ca:         caPath,
				ServerName: "foobar",
				SkipVerify: true,
			},
			wantErr: false,
			verify: func(t require.TestingT, c *tls.Config) {
				require.NotEmpty(t, c.Certificates)
				require.Equal(t, "foobar", c.ServerName, "wrong servername")
				require.True(t, c.InsecureSkipVerify, "InsecureSkipVerify not set")
			},
		},
		"Test 2: CA only config should use CA": {
			conf: &config.TLSConfig{
				Ca: caPath,
			},
			wantErr: false,
			verify: func(t require.TestingT, c *tls.Config) {
				require.NotNil(t, c.RootCAs, "RootCAs should be initialized")
				require.False(t, x509.NewCertPool().Equal(c.RootCAs),
					"CertPool shouldn't be empty, valid CA cert was specified")
				require.False(t, c.InsecureSkipVerify, "InsecureSkipVerify should not be set")
			},
		},
		"Test 3: incorrect CA should not error": {
			conf: &config.TLSConfig{
				Ca: "customca.pem",
			},
			wantErr: true,
		},
		"Test 4: incorrect key path should error": {
			conf: &config.TLSConfig{
				Ca:   caPath,
				Cert: certPath,
				Key:  "badkey",
			},
			wantErr: true,
		},
		"Test 5: incorrect cert path should error": {
			conf: &config.TLSConfig{
				Ca:   caPath,
				Cert: "badcert",
				Key:  keyPath,
			},
			wantErr: true,
		},
		"Test 6: incomplete cert info should error": {
			conf: &config.TLSConfig{
				Key: keyPath,
			},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tlsConfigForCredentials(tt.conf)
			if tt.wantErr {
				require.Error(t, err, "tlsConfigForCredentials(%v)", tt.conf)
				return
			}
			require.NoError(t, err, "tlsConfigForCredentials(%v)", tt.conf)
			if tt.verify != nil {
				tt.verify(t, got)
			}
		})
	}
}
