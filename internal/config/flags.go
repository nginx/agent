// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import "strings"

const (
	ClientTimeoutKey                     = "client_timeout"
	ConfigDirectoriesKey                 = "config_dirs"
	ConfigPathKey                        = "path"
	DataPlaneAPIHostKey                  = "data_plane_api_host"
	DataPlaneAPIPortKey                  = "data_plane_api_port"
	LogLevelKey                          = "log_level"
	LogPathKey                           = "log_path"
	MetricsRootKey                       = "metrics"
	CommandRootKey                       = "command"
	ProcessMonitorMonitoringFrequencyKey = "process_monitor_monitoring_frequency"
	VersionKey                           = "version"
	// Below consts are NOT flag keys.
	OTelExporterRoot     = "otel_exporter"
	GRPCRoot             = "grpc"
	PrometheusSourceRoot = "prometheus_source"
)

var (
	// child flags saved as vars to enable easier prefixing.
	MetricsProduceIntervalKey       = pre(MetricsRootKey) + "produce_interval"
	MetricsOTelExporterKey          = pre(MetricsRootKey) + OTelExporterRoot
	OTelExporterBufferLengthKey     = pre(MetricsOTelExporterKey) + "buffer_length"
	OTelExporterExportRetryCountKey = pre(MetricsOTelExporterKey) + "export_retry_count"
	OTelExporterExportIntervalKey   = pre(MetricsOTelExporterKey) + "export_interval"
	OTelGRPCKey                     = pre(MetricsOTelExporterKey) + GRPCRoot
	OTelGRPCTargetKey               = pre(OTelGRPCKey) + "target"
	OTelGRPCConnTimeoutKey          = pre(OTelGRPCKey) + "connection_timeout"
	OTelGRPCMinConnTimeoutKey       = pre(OTelGRPCKey) + "minimum_connection_timeout"
	OTelGRPCBackoffDelayKey         = pre(OTelGRPCKey) + "backoff_delay"
	PrometheusSrcKey                = pre(MetricsRootKey) + PrometheusSourceRoot
	PrometheusTargetsKey            = pre(PrometheusSrcKey) + "endpoints"
	CommandServerKey                = pre(CommandRootKey) + "server"
	CommandServerHostKey            = pre(CommandServerKey) + "host"
	CommandServerPortKey            = pre(CommandServerKey) + "port"
	CommandServerTypeKey            = pre(CommandServerKey) + "type"
	CommandAuthKey                  = pre(CommandRootKey) + "auth"
	CommandAuthTokenKey             = pre(CommandAuthKey) + "token"
	CommandTLSKey                   = pre(CommandRootKey) + "tls"
	CommandTLSEnableKey             = pre(CommandTLSKey) + "enable"
	CommandTLSCertKey               = pre(CommandTLSKey) + "cert"
	CommandTLSKeyKey                = pre(CommandTLSKey) + "key"
	CommandTLSCaKey                 = pre(CommandTLSKey) + "ca"
	CommandTLSSkipVerifyKey         = pre(CommandTLSKey) + "skip_verify"
)

func pre(prefixes ...string) string {
	joined := strings.Join(prefixes, keyDelimiter)
	return joined + keyDelimiter
}
