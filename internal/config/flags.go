// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"strings"
)

const (
	ClientRootKey                               = "client"
	ConfigDirectoriesKey                        = "config_dirs"
	ConfigPathKey                               = "path"
	CommandRootKey                              = "command"
	DataPlaneConfigRootKey                      = "data_plane_config"
	LogLevelRoot                                = "log"
	MetricsRootKey                              = "metrics"
	VersionKey                                  = "version"
	UUIDKey                                     = "uuid"
	InstanceWatcherMonitoringFrequencyKey       = "watchers_instance_watcher_monitoring_frequency"
	InstanceHealthWatcherMonitoringFrequencyKey = "watchers_instance_health_watcher_monitoring_frequency"
)

var (
	// child flags saved as vars to enable easier prefixing.
	ClientPermitWithoutStreamKey   = pre(ClientRootKey) + "permit_without_stream"
	ClientTimeKey                  = pre(ClientRootKey) + "time"
	ClientTimeoutKey               = pre(ClientRootKey) + "timeout"
	CommandAuthKey                 = pre(CommandRootKey) + "auth"
	CommandAuthTokenKey            = pre(CommandAuthKey) + "token"
	CommandServerHostKey           = pre(CommandServerKey) + "host"
	CommandServerKey               = pre(CommandRootKey) + "server"
	CommandServerPortKey           = pre(CommandServerKey) + "port"
	CommandServerTypeKey           = pre(CommandServerKey) + "type"
	CommandTLSCaKey                = pre(CommandTLSKey) + "ca"
	CommandTLSCertKey              = pre(CommandTLSKey) + "cert"
	CommandTLSKey                  = pre(CommandRootKey) + "tls"
	CommandTLSKeyKey               = pre(CommandTLSKey) + "key"
	CommandTLSServerNameKey        = pre(CommandRootKey) + "server_name"
	CommandTLSSkipVerifyKey        = pre(CommandTLSKey) + "skip_verify"
	NginxReloadMonitoringPeriodKey = pre(DataPlaneConfigRootKey, "nginx") + "reload_monitoring_period"
	NginxTreatWarningsAsErrorsKey  = pre(DataPlaneConfigRootKey, "nginx") + "treat_warnings_as_error"
	LogLevelKey                    = pre(LogLevelRoot) + "level"
	LogPathKey                     = pre(LogLevelRoot) + "path"
	MetricsCollectorConfigPathKey  = pre(MetricsRootKey) + "collector_config_path"
	MetricsCollectorEnabledKey     = pre(MetricsRootKey) + "collector_enabled"
	MetricsCollectorReceiversKey   = pre(MetricsRootKey) + "collector_receivers"
	MetricsOTLPExportURLKey        = pre(MetricsRootKey) + "otlp_export_url"
	MetricsOTLPReceiverURLKey      = pre(MetricsRootKey) + "otlp_receiver_url"
)

func pre(prefixes ...string) string {
	joined := strings.Join(prefixes, KeyDelimiter)
	return joined + KeyDelimiter
}
