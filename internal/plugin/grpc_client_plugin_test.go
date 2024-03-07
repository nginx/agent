// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"testing"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/assert"
)

var grpcAgentConfig *config.Config = &config.Config{
	Command: &config.Command{
		Server: &config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8888,
			Type: "grpc",
		},
	},
}

func TestGrpcClient_NewGrpcClient(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		expected    *GrpcClient
	}{
		{
			"grpc config",
			grpcAgentConfig,
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

func TestGrpcClient_Info(t *testing.T) {
	grpcClient := NewGrpcClient(grpcAgentConfig)
	info := grpcClient.Info()
	assert.Equal(t, "grpc-client", info.Name)
}

func TestGrpcClient_Subscriptions(t *testing.T) {
	grpcClient := NewGrpcClient(grpcAgentConfig)
	subscriptions := grpcClient.Subscriptions()
	assert.Empty(t, subscriptions)
}
