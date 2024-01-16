/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
)

func LoadPlugins(agentConfig *config.Config, slogger *slog.Logger) []bus.Plugin {
	plugins := []bus.Plugin{}

	processMonitor := NewProcessMonitor(&ProcessMonitorParameters{
		MonitoringFrequency: agentConfig.ProcessMonitor.MonitoringFrequency,
	})

	plugins = append(plugins, processMonitor)

	if agentConfig.DataplaneAPI.Host != "" && agentConfig.DataplaneAPI.Port != 0 {
		dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
			Host:   agentConfig.DataplaneAPI.Host,
			Port:   agentConfig.DataplaneAPI.Port,
			Logger: slogger,
		})
		plugins = append(plugins, dataplaneServer)
	}

	return plugins
}
