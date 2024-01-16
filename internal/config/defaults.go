/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import "time"

const (
	ConfigPathKey                              = "path"
	VersionConfigKey                           = "version"
	LogLevelConfigKey                          = "log_level"
	LogPathConfigKey                           = "log_path"
	ProcessMonitorMonitoringFrequencyConfigKey = "process_monitor_monitoring_frequency"
	DataplaneAPIHostConfigKey                  = "dataplane_api_host"
	DataplaneAPIPortConfigKey                  = "dataplane_api_port"
)

var agentFlags = []Registrable{
	&StringFlag{
		Name:         LogLevelConfigKey,
		Usage:        "The desired verbosity level for logging messages from nginx-agent. Available options, in order of severity from highest to lowest, are: panic, fatal, error, info, debug, and trace.",
		DefaultValue: "info",
	},
	&StringFlag{
		Name:  LogPathConfigKey,
		Usage: "The path to output log messages to. If the default path doesn't exist, log messages are output to stdout/stderr.",
	},
	&DurationFlag{
		Name:         ProcessMonitorMonitoringFrequencyConfigKey,
		Usage:        "How often the NGINX Agent will check for process changes.",
		DefaultValue: time.Minute,
	},
	&StringFlag{
		Name:  DataplaneAPIHostConfigKey,
		Usage: "The host used by the Dataplane API.",
	},
	&IntFlag{
		Name:  DataplaneAPIPortConfigKey,
		Usage: "The desired port to use for NGINX Agent to expose for HTTP traffic.",
	},
}
