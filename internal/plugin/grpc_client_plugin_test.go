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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAgentConfig() *config.Config {
	return &config.Config{
		DataPlaneAPI: &config.DataPlaneAPI{
			Host: "127.0.0.1",
			Port: 8989,
		},
		Command: &config.Command{
			Server: &config.ServerConfig{
				Host: "127.0.0.1",
				Port: 8888,
				Type: "grpc",
			},
		},
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func TestGrpcClient_NewGrpcClient(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		expected    *GrpcClient
	}{
		{
			"grpc config",
			getAgentConfig(),
			&GrpcClient{},
		},
		{
			"not grpc type",
			&config.Config{
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 8888,
						Type: "http",
					},
				},
			},
			nil,
		},
		{
			"nil client",
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	agentConfig := getAgentConfig()
	agentConfig.Command.Server.Host = "saasdkldsj"

	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{client})
	require.NoError(t, err)

	err = client.Init(messagePipe)
	require.NoError(t, err)
	assert.Contains(t, err.Error(), "no such host")
}

func TestGrpcClient_Info(t *testing.T) {
	grpcClient := NewGrpcClient(getAgentConfig())
	info := grpcClient.Info()
	assert.Equal(t, "grpc-client", info.Name)
}

func TestGrpcClient_Subscriptions(t *testing.T) {
	grpcClient := NewGrpcClient(getAgentConfig())
	subscriptions := grpcClient.Subscriptions()
	assert.Empty(t, subscriptions)
}

func TestGrpcClient_Process(t *testing.T) {
	agentConfig := getAgentConfig()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	mockMessage := &bus.Message{
		Topic: bus.InstanceConfigUpdateRequestTopic,
		Data:  nil,
	}
	client.Process(mockMessage)
	// add better assertions when the process function does something
	assert.Nil(t, client.messagePipe)
}

func TestGetDialOptions(t *testing.T) {
	agentConfig := getAgentConfig()
	client := NewGrpcClient(agentConfig)
	assert.NotNil(t, client)

	options := client.getDialOptions()

	assert.NotNil(t, options)

	// Ensure the expected number of dial options, will change over time
	assert.Len(t, options, 6)
}
