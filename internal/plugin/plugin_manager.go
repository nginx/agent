// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
)

func LoadPlugins(agentConfig *config.Config, slogger *slog.Logger) []bus.Plugin {
	plugins := make([]bus.Plugin, 0)

	plugins = addProcessMonitor(agentConfig, plugins)
	plugins = addResourceMonitor(agentConfig, plugins)

	configPlugin := NewConfig(agentConfig)

	plugins = addMetrics(agentConfig, slogger, plugins)
	plugins = append(plugins, configPlugin)

	if isGrpcClientConfigured(agentConfig) {
		grpcClient := NewGrpcClient(agentConfig)
		plugins = append(plugins, grpcClient)
	}

	return plugins
}

func addMetrics(agentConfig *config.Config, slogger *slog.Logger, plugins []bus.Plugin) []bus.Plugin {
	if agentConfig.Metrics != nil {
		metrics, err := NewMetrics(agentConfig)
		if err != nil {
			slogger.Error("Failed to initialize metrics plugin", "error", err)
		} else {
			plugins = append(plugins, metrics)
		}
	}

	return plugins
}

func addResourceMonitor(agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	instanceMonitor := NewResource(agentConfig)
	plugins = append(plugins, instanceMonitor)

	return plugins
}

func addProcessMonitor(agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	if agentConfig.ProcessMonitor != nil && agentConfig.ProcessMonitor.MonitoringFrequency != 0 {
		processMonitor := NewProcessMonitor(agentConfig)
		plugins = append(plugins, processMonitor)
	}

	return plugins
}

func isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Type == "grpc"
}
