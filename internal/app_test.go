// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/collector"
	"github.com/nginx/agent/v3/internal/command"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/file"
	"github.com/nginx/agent/v3/internal/resource"
	"github.com/nginx/agent/v3/internal/watcher"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	app := NewApp("1234", "1.2.3")

	err := app.Run(context.Background())

	require.NoError(t, err)
}

func TestLoadPlugins(t *testing.T) {
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
				&watcher.Watcher{},
			},
		}, {
			name: "Test 2: Load file and command plugins",
			input: &config.Config{
				Command: &config.Command{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 443,
						Type: config.Grpc,
					},
				},
			},
			expected: []bus.Plugin{
				&resource.Resource{},
				&command.CommandPlugin{},
				&file.FilePlugin{},
				&watcher.Watcher{},
			},
		}, {
			name: "Test 3: Load metrics collector plugin",
			input: &config.Config{
				Collector: &config.Collector{},
			},
			expected: []bus.Plugin{
				&resource.Resource{},
				&collector.Collector{},
				&watcher.Watcher{},
			},
		},
	}

	app := NewApp("test", "test")

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			t.Logf("running test %s", test.name)
			result := app.loadPlugins(test.input)
			assert.Equal(tt, len(test.expected), len(result))
			for i, expectedPlugin := range test.expected {
				assert.IsType(tt, expectedPlugin, result[i])
			}
		})
	}
}
