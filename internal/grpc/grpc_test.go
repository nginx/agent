// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"fmt"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
)

func TestGrpcClient_GetDialOptions(t *testing.T) {
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
			8,
			false,
		},
		{
			"Test 2: DialOptions mTLS",
			types.GetAgentConfig(),
			8,
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
			8,
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
			7,
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
			7,
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

	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			if tt.createCerts {
				tmpDir := t.TempDir()
				// not mtls scripts
				key, cert := helpers.GenerateSelfSignedCert(t)
				_, ca := helpers.GenerateSelfSignedCert(t)

				keyContents := helpers.Cert{Name: keyFileName, Type: certificateType, Contents: key}
				certContents := helpers.Cert{Name: certFileName, Type: privateKeyType, Contents: cert}
				caContents := helpers.Cert{Name: caFileName, Type: certificateType, Contents: ca}

				helpers.WriteCertFiles(t, tmpDir, keyContents)
				helpers.WriteCertFiles(t, tmpDir, certContents)
				helpers.WriteCertFiles(t, tmpDir, caContents)

				tt.agentConfig.Command.TLS.Cert = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, certFileName)
				tt.agentConfig.Command.TLS.Key = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, keyFileName)
				tt.agentConfig.Command.TLS.Ca = fmt.Sprintf("%s%s%s", tmpDir, pathSeparator, caFileName)
			}

			options := GetDialOptions(tt.agentConfig)
			assert.NotNil(ttt, options)
			assert.Len(ttt, options, tt.expected)
		})
	}
}
