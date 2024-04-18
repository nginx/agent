// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"

	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcClient_NewGrpcClient(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		expected    *GrpcClient
	}{
		{
			"Test 1: GRPC type specified in config",
			types.GetAgentConfig(),
			&GrpcClient{},
		},
		{
			"Test 2: GRPC type not specified in config",
			&config.Config{
				Command: types.GetAgentConfig().Command,
				Common:  types.GetAgentConfig().Common,
				Client:  types.GetAgentConfig().Client,
			},
			nil,
		},
		{
			"Test 3: nil client, nil settings",
			nil,
			nil,
		},
		{
			"Test 4: nil client settings",
			&config.Config{
				Command: types.GetAgentConfig().Command,
				Common:  types.GetAgentConfig().Common,
			},
			nil,
		},
		{
			"Test 5: nil common settings",
			&config.Config{
				Command: types.GetAgentConfig().Command,
				Client:  types.GetAgentConfig().Client,
			},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			grpcClient := NewGrpcClient(test.agentConfig)

			if grpcClient == nil {
				assert.Equal(t, test.expected, grpcClient)
			} else {
				assert.IsType(t, test.expected, grpcClient)
			}
		})
	}
}

func TestGrpcClient_Init(t *testing.T) {
	tests := []struct {
		name          string
		agentConfig   *config.Config
		server        string
		expectedError bool
		errorMessage  string
	}{
		{
			"Test 1: GRPC type specified in config",
			types.GetAgentConfig(),
			"incorrect-server",
			true,
			"connection error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			ctx := context.Background()
			test.agentConfig.Command.Server.Host = test.server

			client := NewGrpcClient(test.agentConfig)
			assert.NotNil(tt, client)

			messagePipe := bus.NewMessagePipe(10)
			err := messagePipe.Register(1, []bus.Plugin{client})
			require.NoError(tt, err)

			err = client.Init(ctx, messagePipe)
			if test.expectedError {
				assert.Contains(tt, err.Error(), test.errorMessage)
			} else {
				require.NoError(tt, err)
			}
		})
	}
}

func TestGrpcClient_Info(t *testing.T) {
	grpcClient := NewGrpcClient(types.GetAgentConfig())
	info := grpcClient.Info()
	assert.Equal(t, "grpc-client", info.Name)
}

func TestGrpcClient_Subscriptions(t *testing.T) {
	grpcClient := NewGrpcClient(types.GetAgentConfig())
	subscriptions := grpcClient.Subscriptions()
	assert.Len(t, subscriptions, 2)
	assert.Equal(t, bus.GrpcConnectedTopic, subscriptions[0])
	assert.Equal(t, bus.ResourceTopic, subscriptions[1])
}

func TestGrpcClient_Process_ResourceTopic(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	expected := protos.GetHostResource()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	fakeCommandServiceClient := &v1fakes.FakeCommandServiceClient{}

	client.commandServiceClient = fakeCommandServiceClient
	client.isConnected.Store(true)

	mockMessage := &bus.Message{
		Topic: bus.ResourceTopic,
		Data:  expected,
	}
	client.Process(ctx, mockMessage)

	assert.True(t, client.isConnected.Load())
}

func TestGrpcClient_Close(t *testing.T) {
	serverMockLock := sync.Mutex{}
	tests := []struct {
		name         string
		agentConfig  *config.Config
		errorMessage string
		createCerts  bool
	}{
		{
			"Test 1: GRPC can't connect, invalid token",
			types.GetAgentConfig(),
			"invalid token",
			false,
		},
		{
			"Test 2: GRPC can connect, insecure",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 2,
						Type: "grpc",
					},
					Auth: types.GetAgentConfig().Command.Auth,
					TLS:  types.GetAgentConfig().Command.TLS,
				},
				Common: types.GetAgentConfig().Common,
			},
			"",
			false,
		},
		{
			"Test 3: GRPC can connect, no auth token",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 4,
						Type: "grpc",
					},
					TLS: types.GetAgentConfig().Command.TLS,
				},
				Common: types.GetAgentConfig().Common,
			},
			"",
			false,
		},
		{
			"Test 4: GRPC can connect, no tls, no auth token",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 6,
						Type: "grpc",
					},
				},
				Common: types.GetAgentConfig().Common,
			},
			"",
			false,
		},
		{
			"Test 5: GRPC can connect with tls, no auth token",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 8,
						Type: "grpc",
					},
					TLS: types.GetAgentConfig().Command.TLS,
				},
				Common: types.GetAgentConfig().Common,
			},
			"",
			true,
		},
		{
			"Test 6: GRPC can't connect, context deadline exceeded",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 10,
						Type: "grpc",
					},
					Auth: types.GetAgentConfig().Command.Auth,
					TLS:  types.GetAgentConfig().Command.TLS,
				},
				Common: &config.CommonSettings{
					MaxElapsedTime: 1 * time.Microsecond,
				},
			},
			"context deadline exceeded",
			false,
		},
		{
			"Test 7: GRPC can connect tls enabled",
			&config.Config{
				Client: types.GetAgentConfig().Client,
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: types.GetAgentConfig().Command.Server.Port + 12,
						Type: "grpc",
					},
					Auth: types.GetAgentConfig().Command.Auth,
					TLS:  types.GetAgentConfig().Command.TLS,
				},
				Common: &config.CommonSettings{
					MaxElapsedTime: 1 * time.Microsecond,
				},
			},
			"",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			address := fmt.Sprintf(
				"%s:%d",
				test.agentConfig.Command.Server.Host,
				test.agentConfig.Command.Server.Port+1,
			)

			if test.createCerts {
				tmpDir := tt.TempDir()
				key, cert := helpers.GenerateSelfSignedCert(tt)

				keyContents := helpers.Cert{Name: "key.pem", Type: "RSA PRIVATE KEY", Contents: key}
				certContents := helpers.Cert{Name: "cert.pem", Type: "CERTIFICATE", Contents: cert}

				keyFile := helpers.WriteCertFiles(tt, tmpDir, keyContents)
				certFile := helpers.WriteCertFiles(tt, tmpDir, certContents)

				test.agentConfig.Command.TLS.Key = keyFile
				test.agentConfig.Command.TLS.Cert = certFile
			}

			serverMockLock.Lock()
			server, err := mockGrpc.NewMockManagementServer(address, test.agentConfig)
			require.NoError(tt, err)
			serverMockLock.Unlock()

			client := NewGrpcClient(test.agentConfig)
			assert.NotNil(tt, client)

			messagePipe := bus.NewMessagePipe(100)
			err = messagePipe.Register(100, []bus.Plugin{client})
			require.NoError(tt, err)

			err = client.Init(context.Background(), messagePipe)
			if err == nil {
				require.NoError(tt, err)
			} else {
				assert.Contains(tt, err.Error(), test.errorMessage)
			}

			err = client.Close(context.Background())
			require.NoError(tt, err)

			defer server.Stop()
		})
	}
}
