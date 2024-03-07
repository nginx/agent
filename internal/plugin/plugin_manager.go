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

	processMonitor := NewProcessMonitor(agentConfig)
	instanceMonitor := NewInstance()

	configPlugin := NewConfig(agentConfig)

	if agentConfig.Metrics != nil {
		metrics, err := NewMetrics(*agentConfig)
		if err != nil {
			slogger.Error("Failed to initialize metrics plugin", "error", err)
		} else {
			plugins = append(plugins, metrics)
		}
	}

	plugins = append(plugins, processMonitor, instanceMonitor, configPlugin)

	if isGrpcClientConfigured(agentConfig) {
		grpcClient := NewGrpcClient(agentConfig)
		plugins = append(plugins, grpcClient)
	}

	if agentConfig.DataPlaneAPI.Host != "" && agentConfig.DataPlaneAPI.Port != 0 {
		dataPlaneServer := NewDataPlaneServer(agentConfig, slogger)
		plugins = append(plugins, dataPlaneServer)
	}

	return plugins
}

func isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Host != "" &&
		agentConfig.Command.Server.Port != 0 &&
		agentConfig.Command.Server.Type == "grpc"
}
