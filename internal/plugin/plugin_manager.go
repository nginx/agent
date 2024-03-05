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

	grpcClient := NewGrpcClient(nil, nil)

	plugins = append(plugins, processMonitor, instanceMonitor, configPlugin, grpcClient)

	if agentConfig.DataPlaneAPI.Host != "" && agentConfig.DataPlaneAPI.Port != 0 {
		dataPlaneServer := NewDataPlaneServer(agentConfig, slogger)
		plugins = append(plugins, dataPlaneServer)
	}

	return plugins
}
