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

	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
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
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 8888,
						Type: "http",
					},
				},
				Common: &config.CommonSettings{
					InitialInterval:     100 * time.Microsecond,
					MaxInterval:         1000 * time.Microsecond,
					MaxElapsedTime:      10 * time.Millisecond,
					RandomizationFactor: 0.1,
					Multiplier:          0.2,
				},
				Client: &config.Client{
					Timeout:             100 * time.Microsecond,
					Time:                200 * time.Microsecond,
					PermitWithoutStream: false,
				},
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
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 8888,
						Type: "http",
					},
				},
				Common: &config.CommonSettings{
					InitialInterval:     100 * time.Microsecond,
					MaxInterval:         1000 * time.Microsecond,
					MaxElapsedTime:      10 * time.Millisecond,
					RandomizationFactor: 0.1,
					Multiplier:          0.2,
				},
			},
			nil,
		},
		{
			"Test 5: nil common settings",
			&config.Config{
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 8888,
						Type: "grpc",
					},
				},
				Client: &config.Client{
					Timeout:             100 * time.Microsecond,
					Time:                200 * time.Microsecond,
					PermitWithoutStream: false,
				},
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

			resource := &v1.Resource{
				Id:        "123",
				Instances: []*v1.Instance{},
				Info: &v1.Resource_ContainerInfo{
					ContainerInfo: &v1.ContainerInfo{
						Id: "f43f5eg54g54g54",
					},
				},
			}

			mockReourceService := &servicefakes.FakeResourceServiceInterface{}
			mockReourceService.GetResourceReturns(resource)

			client := NewGrpcClient(test.agentConfig)
			client.resourceService = mockReourceService
			assert.NotNil(tt, client)

			messagePipe := bus.NewMessagePipe(100)
			err := messagePipe.Register(100, []bus.Plugin{client})
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
	assert.Len(t, subscriptions, 3)
	assert.Equal(t, bus.InstancesTopic, subscriptions[0])
	assert.Equal(t, bus.GrpcConnectedTopic, subscriptions[1])
	assert.Equal(t, bus.InstanceConfigUpdateStatusTopic, subscriptions[2])
}

func TestGrpcClient_Process_InstancesTopic(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	fakeCommandServiceClient := &v1fakes.FakeCommandServiceClient{}
	fakeCommandServiceClient.UpdateDataPlaneStatusReturns(&v1.UpdateDataPlaneStatusResponse{}, nil)

	client.commandServiceClient = fakeCommandServiceClient
	client.isConnected.Store(true)

	mockMessage := &bus.Message{
		Topic: bus.InstancesTopic,
		Data:  []*v1.Instance{},
	}
	client.Process(ctx, mockMessage)

	assert.Equal(t, 1, fakeCommandServiceClient.UpdateDataPlaneStatusCallCount())
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
