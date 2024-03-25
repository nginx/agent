// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"
	"time"

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
						Type: "http",
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

func TestGrpcClient_InitWithInvalidServerAddr(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	agentConfig.Command.Server.Host = "saasdkldsj"

	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	messagePipe := bus.NewMessagePipe(100)
	err := messagePipe.Register(100, []bus.Plugin{client})
	require.NoError(t, err)

	err = client.Init(ctx, messagePipe)
	assert.Contains(t, err.Error(), "connection error")
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
		Topic: bus.InstanceConfigUpdateRequestTopic,
		Data:  nil,
	}
	client.Process(ctx, mockMessage)
	// add better assertions when the process function does something
	assert.Nil(t, client.messagePipe)
}

func TestGetDialOptions(t *testing.T) {
	agentConfig := types.GetAgentConfig()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	options := client.getDialOptions()

	assert.NotNil(t, options)

	// Ensure the expected number of dial options, will change over time
	assert.Len(t, options, 7)
}
