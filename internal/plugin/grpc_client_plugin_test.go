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

	"github.com/cenkalti/backoff/v4"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

type TestError struct{}

func (z TestError) Error() string {
	return "Test"
}

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
		name        string
		agentConfig *config.Config
		server      string
	}{
		{
			"Test 1: GRPC type specified in config",
			types.GetAgentConfig(),
			"server.com",
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
			require.NoError(tt, err)
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
	assert.Equal(t, bus.ResourceUpdateTopic, subscriptions[0])
	assert.Equal(t, bus.InstanceHealthTopic, subscriptions[1])
}

func TestGrpcClient_Process(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	expected := protos.GetHostResource()
	client := NewGrpcClient(agentConfig)
	client.messagePipe = &bus.FakeMessagePipe{}
	assert.NotNil(t, client)

	tests := []struct {
		name                           string
		message                        *bus.Message
		updateDataPlaneStatusCallCount int
		updateDataPlaneHealthCallCount int
	}{
		{
			name: "Test 1: ResourceTopic",
			message: &bus.Message{
				Topic: bus.ResourceUpdateTopic,
				Data:  expected,
			},
			updateDataPlaneStatusCallCount: 1,
			updateDataPlaneHealthCallCount: 0,
		}, {
			name: "Test 2: InstanceHealthTopic",
			message: &bus.Message{
				Topic: bus.InstanceHealthTopic,
				Data:  protos.GetInstanceHealths(),
			},
			updateDataPlaneStatusCallCount: 0,
			updateDataPlaneHealthCallCount: 1,
		}, {
			name: "Test 3: UnknownTopic",
			message: &bus.Message{
				Topic: "unknown",
				Data:  nil,
			},
			updateDataPlaneStatusCallCount: 0,
			updateDataPlaneHealthCallCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeCommandServiceClient := &v1fakes.FakeCommandServiceClient{}
			client.commandServiceClient = fakeCommandServiceClient

			client.Process(ctx, test.message)

			assert.True(t, client.isConnected.Load())
			assert.Equal(
				t,
				test.updateDataPlaneStatusCallCount,
				fakeCommandServiceClient.UpdateDataPlaneStatusCallCount(),
			)
			assert.Equal(
				t,
				test.updateDataPlaneHealthCallCount,
				fakeCommandServiceClient.UpdateDataPlaneHealthCallCount(),
			)
		})
	}
}

func TestGrpcClient_sendDataPlaneHealthUpdate(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	instances := protos.GetInstanceHealths()

	client := NewGrpcClient(agentConfig)
	client.messagePipe = &bus.FakeMessagePipe{}
	assert.NotNil(t, client)
	fakeCommandServiceClient := &v1fakes.FakeCommandServiceClient{}

	client.commandServiceClient = fakeCommandServiceClient
	err := client.createConnection(ctx, protos.GetHostResource())
	require.NoError(t, err)

	healthErr := client.sendDataPlaneHealthUpdate(ctx, instances)
	require.NoError(t, healthErr)
}

func TestGrpcClient_ProcessRequest(t *testing.T) {
	ctx := context.Background()

	agentConfig := types.GetAgentConfig()
	client := NewGrpcClient(agentConfig)
	pipe := &bus.FakeMessagePipe{}
	client.messagePipe = pipe
	assert.NotNil(t, client)

	tests := []struct {
		name     string
		request  *v1.ManagementPlaneRequest
		expected *bus.Message
	}{
		{
			name: "Test 1: Config Apply Request",
			request: &v1.ManagementPlaneRequest{
				Request: &v1.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: &v1.ConfigApplyRequest{
						ConfigVersion: protos.CreateConfigVersion(),
					},
				},
			},
			expected: &bus.Message{
				Topic: bus.InstanceConfigUpdateRequestTopic,
				Data: &v1.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: &v1.ConfigApplyRequest{
						ConfigVersion: protos.CreateConfigVersion(),
					},
				},
			},
		},
		{
			name: "Test 2: Invalid Request",
			request: &v1.ManagementPlaneRequest{
				Request: &v1.ManagementPlaneRequest_ActionRequest{},
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			client.ProcessRequest(ctx, test.request)
			pipe.GetMessages()
		})
	}
}

func TestGrpcClient_Close(t *testing.T) {
	ctx := context.Background()
	serverMockLock := sync.Mutex{}
	configDirectory := ""

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
			server, err := mockGrpc.NewMockManagementServer(address, test.agentConfig, &configDirectory)
			require.NoError(tt, err)
			defer server.Stop()
			serverMockLock.Unlock()

			client := NewGrpcClient(test.agentConfig)
			assert.NotNil(tt, client)

			messagePipe := bus.NewFakeMessagePipe()

			err = client.Init(ctx, messagePipe)
			if err == nil {
				require.NoError(tt, err)
			} else {
				assert.Contains(tt, err.Error(), test.errorMessage)
			}

			err = client.Close(ctx)
			require.NoError(tt, err)
		})
	}
}

func TestGrpcClient_validateGrpcError(t *testing.T) {
	result := validateGrpcError(TestError{})
	assert.IsType(t, TestError{}, result)

	result = validateGrpcError(status.Errorf(codes.InvalidArgument, "error"))
	assert.IsType(t, &backoff.PermanentError{}, result)
}

func TestGrpcConfigClient_GetOverview(t *testing.T) {
	ctx := context.Background()
	fileServiceClient := v1fakes.FakeFileServiceClient{}
	overviewResponse, err := protos.CreateGetOverviewResponse()
	fileResponse := protos.CreateGetFileResponse([]byte("location /test {\n    return 200 \"Test location\\n\";\n}"))
	fileServiceClient.GetOverviewReturns(overviewResponse, nil)
	fileServiceClient.GetFileReturns(fileResponse, nil)
	require.NoError(t, err)

	client := NewGrpcClient(types.GetAgentConfig())
	assert.NotNil(t, client)

	client.fileServiceClient = &fileServiceClient

	gcc := GrpcConfigClient{
		grpcOverviewFn:    client.GetFileOverview,
		grpFileContentsFn: client.GetFileContents,
	}
	req := protos.CreateGetOverviewRequest()

	resp, err := gcc.GetOverview(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, overviewResponse.GetOverview(), resp)
}

func TestGrpcConfigClient_GetFile(t *testing.T) {
	ctx := context.Background()
	fileServiceClient := v1fakes.FakeFileServiceClient{}
	overviewResponse, err := protos.CreateGetOverviewResponse()
	fileResponse := protos.CreateGetFileResponse([]byte("location /test {\n    return 200 \"Test location\\n\";\n}"))
	fileServiceClient.GetOverviewReturns(overviewResponse, nil)
	fileServiceClient.GetFileReturns(fileResponse, nil)
	require.NoError(t, err)

	client := NewGrpcClient(types.GetAgentConfig())
	assert.NotNil(t, client)

	client.fileServiceClient = &fileServiceClient

	gcc := GrpcConfigClient{
		grpcOverviewFn:    client.GetFileOverview,
		grpFileContentsFn: client.GetFileContents,
	}
	req, err := protos.CreateGetFileRequest("nginx.conf")
	require.NoError(t, err)

	resp, err := gcc.GetFile(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, fileResponse.GetContents(), resp)
}
