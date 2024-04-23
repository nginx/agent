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
	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			reply:           protos.GetNginxOssInstance(),
			isErrorExpected: true,
		},
		{
			name:            "Test 2: Invalid reply type",
			request:         protos.GetNginxOssInstance(),
			reply:           "invalid",
			isErrorExpected: true,
		},
		{
			name:            "Test 3: Valid request & reply types",
			request:         protos.GetNginxOssInstance(),
			reply:           protos.GetNginxOssInstance(),
			isErrorExpected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			validationError := interceptor(ctx, "", test.request, test.reply, nil, invoker, nil)
			tt.Log(validationError)
			assert.Equal(tt, test.isErrorExpected, validationError != nil)
		})
	}
}
