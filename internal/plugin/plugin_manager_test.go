// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"testing"

	"github.com/nginx/agent/v3/internal/metrics/collector"
	"github.com/nginx/agent/v3/internal/resource"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/watcher"
	"github.com/stretchr/testify/assert"
)

func TestLoadPLugins(t *testing.T) {
	tests := []struct {
		name     string
		input    *config.Config
		expected []bus.Plugin
	}{
		{
			name:  "Test 1: Load plugins",
			input: &config.Config{},
			expected: []bus.Plugin{
				&resource.Resource{},
				&Config{},
				&watcher.Watcher{},
			},
		}, {
			name: "Test 2: Load grpc client plugin",
			input: &config.Config{
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 443,
						Type: "grpc",
					},
				},
			},
			expected: []bus.Plugin{
				&resource.Resource{},
				&Config{},
				&GrpcClient{},
				&watcher.Watcher{},
			},
		}, {
			name: "Test 3: Load metrics collector plugin",
			input: &config.Config{
				Metrics: &config.Metrics{
					Collector: true,
				},
			},
			expected: []bus.Plugin{
				&resource.Resource{},
				&Config{},
				&collector.Collector{},
				&watcher.Watcher{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			t.Logf("running test %s", test.name)
			result := LoadPlugins(test.input)
			assert.Equal(tt, len(test.expected), len(result))
			for i, expectedPlugin := range test.expected {
				assert.IsType(tt, expectedPlugin, result[i])
			}
		})
	}
}
