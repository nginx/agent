// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"strings"
	"time"
)

const (
	ConfigDirectoriesKey                          = "config_dirs"
	ConfigPathKey                                 = "path"
	CommandRootKey                                = "command"
	DataPlaneConfigNginxReloadMonitoringPeriodKey = "data_plane_config_nginx_reload_monitoring_period"
	DataPlaneConfigNginxTreatWarningsAsErrorsKey  = "data_plane_config_nginx_treat_warnings_as_error"
	LogLevelKey                                   = "log_level"
	LogPathKey                                    = "log_path"
	MetricsRootKey                                = "metrics"
	VersionKey                                    = "version"
	UUIDKey                                       = "uuid"
	InstanceWatcherMonitoringFrequencyKey         = "watchers_instance_watcher_monitoring_frequency"
	InstanceHealthWatcherMonitoringFrequencyKey   = "watchers_instance_health_watcher_monitoring_frequency"

	// Below consts are NOT flag keys.
	PrometheusSourceRoot = "prometheus_source"

	DefaultDataPlaneConfigNginxReloadMonitoringPeriod = 10 * time.Second

	ClientRootKey = "client"
)

var (
	// child flags saved as vars to enable easier prefixing.
	MetricsCollectorKey          = pre(MetricsRootKey) + "collector"
	CommandServerKey             = pre(CommandRootKey) + "server"
	CommandServerHostKey         = pre(CommandServerKey) + "host"
	CommandServerPortKey         = pre(CommandServerKey) + "port"
	CommandServerTypeKey         = pre(CommandServerKey) + "type"
	CommandAuthKey               = pre(CommandRootKey) + "auth"
	CommandAuthTokenKey          = pre(CommandAuthKey) + "token"
	CommandTLSKey                = pre(CommandRootKey) + "tls"
	CommandTLSCertKey            = pre(CommandTLSKey) + "cert"
	CommandTLSKeyKey             = pre(CommandTLSKey) + "key"
	CommandTLSCaKey              = pre(CommandTLSKey) + "ca"
	CommandTLSSkipVerifyKey      = pre(CommandTLSKey) + "skip_verify"
	CommandTLSServerNameKey      = pre(CommandRootKey) + "server_name"
	ClientTimeoutKey             = pre(ClientRootKey) + "timeout"
	ClientTimeKey                = pre(ClientRootKey) + "time"
	ClientPermitWithoutStreamKey = pre(ClientRootKey) + "permit_without_stream"
)

func pre(prefixes ...string) string {
	joined := strings.Join(prefixes, KeyDelimiter)
	return joined + KeyDelimiter
}
