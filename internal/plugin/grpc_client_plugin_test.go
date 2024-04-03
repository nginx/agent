// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/helpers"

	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var grpcServerMutex = &sync.Mutex{}

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
					InitialInterval: 100 * time.Microsecond,
					MaxInterval:     1000 * time.Microsecond,
					MaxElapsedTime:  10 * time.Millisecond,
					Jitter:          0.1,
					Multiplier:      0.2,
				},
				Client: &config.Client{
					Timeout:      100 * time.Microsecond,
					Time:         200 * time.Microsecond,
					PermitStream: false,
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
					InitialInterval: 100 * time.Microsecond,
					MaxInterval:     1000 * time.Microsecond,
					MaxElapsedTime:  10 * time.Millisecond,
					Jitter:          0.1,
					Multiplier:      0.2,
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
					Timeout:      100 * time.Microsecond,
					Time:         200 * time.Microsecond,
					PermitStream: false,
				},
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			grpcClient := NewGrpcClient(tt.agentConfig)

			if grpcClient == nil {
				assert.Equal(t, tt.expected, grpcClient)
			} else {
				assert.IsType(t, tt.expected, grpcClient)
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

	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			ctx := context.Background()
			tt.agentConfig.Command.Server.Host = tt.server
			client := NewGrpcClient(tt.agentConfig)
			assert.NotNil(t, client)

			messagePipe := bus.NewMessagePipe(100)
			err := messagePipe.Register(100, []bus.Plugin{client})
			require.NoError(t, err)

			err = client.Init(ctx, messagePipe)
			if tt.expectedError {
				assert.Contains(ttt, err.Error(), tt.errorMessage)
			} else {
				require.NoError(ttt, err)
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
	assert.Len(t, subscriptions, 1)
	assert.Equal(t, bus.GrpcConnectedTopic, subscriptions[0])
}

func TestGrpcClient_Process(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	mockMessage := &bus.Message{
		Topic: bus.GrpcConnectedTopic,
		Data:  nil,
	}
	client.Process(ctx, mockMessage)
	// add better assertions when the process function does something
	assert.Nil(t, client.messagePipe)
}

func TestGrpcClient_Close(t *testing.T) {
	ctx := context.Background()
	agentConfig := &config.Config{
		Client: types.GetAgentConfig().Client,
		Command: &config.Command{
			Server: &config.ServerConfig{
				Host: "127.0.0.1",
				Port: 9999,
				Type: "grpc",
			},
		},
		Common: types.GetAgentConfig().Common,
	}

	server, err := startMockGrpcServer(
		fmt.Sprintf(
			"%s:%d",
			agentConfig.Command.Server.Host,
			agentConfig.Command.Server.Port),
		agentConfig)

	require.NoError(t, err)

	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	messagePipe := bus.NewMessagePipe(100)
	err = messagePipe.Register(100, []bus.Plugin{client})
	require.NoError(t, err)

	err = client.Init(ctx, messagePipe)
	require.NoError(t, err)

	err = client.Close(context.Background())
	require.NoError(t, err)

	stopMockCommandServer(server)
}

func TestGrpcClient_createConnection(t *testing.T) {
	tests := []struct {
		name         string
		agentConfig  *config.Config
		errorMessage string
		createCerts  bool
	}{
		{
			"Test 1: GRPC can't connect, invalid token",
			types.GetAgentConfig(),
			"cannot send secure credentials on an insecure connection",
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
	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			address := fmt.Sprintf("%s:%d", tt.agentConfig.Command.Server.Host, tt.agentConfig.Command.Server.Port+1)

			if tt.createCerts {
				tt.agentConfig.Command.TLS.Enable = tt.createCerts
				tmpDir := t.TempDir()
				key, cert := helpers.GenerateSelfSignedCert(ttt)

				keyContents := helpers.Cert{Name: "key.pem", Type: "RSA PRIVATE KEY", Contents: key}
				certContents := helpers.Cert{Name: "cert.pem", Type: "CERTIFICATE", Contents: cert}

				keyFile := helpers.WriteCertFiles(t, tmpDir, keyContents)
				certFile := helpers.WriteCertFiles(t, tmpDir, certContents)

				tt.agentConfig.Command.TLS.Key = keyFile
				tt.agentConfig.Command.TLS.Cert = certFile
			}

			server, err := startMockGrpcServer(address, tt.agentConfig)
			require.NoError(ttt, err)

			client := NewGrpcClient(tt.agentConfig)
			assert.NotNil(t, client)

			messagePipe := bus.NewMessagePipe(100)
			err = messagePipe.Register(100, []bus.Plugin{client})
			require.NoError(t, err)

			err = client.Init(context.Background(), messagePipe)
			if err == nil {
				require.NoError(ttt, err)
			} else {
				assert.Contains(ttt, err.Error(), tt.errorMessage)
			}

			stopMockCommandServer(server)
		})
	}
}

func startMockGrpcServer(
	address string,
	agentConfig *config.Config,
) (*mockGrpc.MockManagementServer, error) {
	grpcServerMutex.Lock()
	defer grpcServerMutex.Unlock()

	return mockGrpc.NewMockManagementServer(address, agentConfig)
}

func stopMockCommandServer(server *mockGrpc.MockManagementServer) {
	grpcServerMutex.Lock()
	defer grpcServerMutex.Unlock()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		signal.Stop(sigs)
		server.GrpcServer.Stop()
		time.Sleep(200 * time.Millisecond)
		done <- true
	}()

	<-done
	server.GrpcServer.GracefulStop()
	time.Sleep(1 * time.Second)
}
