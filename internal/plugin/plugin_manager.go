// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/internal/metrics/collector"
	"github.com/nginx/agent/v3/internal/resource"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/watcher"
)

func LoadPlugins(agentConfig *config.Config) []bus.Plugin {
	plugins := make([]bus.Plugin, 0)

	plugins = addResourcePlugin(plugins)

	configPlugin := NewConfig(agentConfig)

	plugins = append(plugins, configPlugin)

	if isGrpcClientConfigured(agentConfig) {
		grpcClient := NewGrpcClient(agentConfig)
		plugins = append(plugins, grpcClient)
	}

	plugins = addCollector(agentConfig, plugins)
	plugins = append(plugins, watcher.NewWatcher(agentConfig))

	return plugins
}

func addResourcePlugin(plugins []bus.Plugin) []bus.Plugin {
	resourcePlugin := resource.NewResource()
	plugins = append(plugins, resourcePlugin)

	return plugins
}

func isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Type == "grpc"
}

func addCollector(agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	if agentConfig.Metrics != nil && agentConfig.Metrics.Collector {
		oTelCollector, err := collector.NewCollector(agentConfig)
		if err == nil {
			plugins = append(plugins, oTelCollector)
		} else {
			slog.Error("Failed to initialize collector plugin", "error", err)
		}
	}

	return plugins
}
