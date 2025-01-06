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
	AllowedDirectoriesKey                       = "allowed_directories"
	ConfigPathKey                               = "path"
	CommandRootKey                              = "command"
	DataPlaneConfigRootKey                      = "data_plane_config"
	LogLevelRootKey                             = "log"
	CollectorRootKey                            = "collector"
	VersionKey                                  = "version"
	UUIDKey                                     = "uuid"
	FeaturesKey                                 = "features"
	InstanceWatcherMonitoringFrequencyKey       = "watchers_instance_watcher_monitoring_frequency"
	InstanceHealthWatcherMonitoringFrequencyKey = "watchers_instance_health_watcher_monitoring_frequency"
	FileWatcherMonitoringFrequencyKey           = "watchers_file_watcher_monitoring_frequency"
)

var (
	// child flags saved as vars to enable easier prefixing.
	ClientPermitWithoutStreamKey                = pre(ClientRootKey) + "permit_without_stream"
	ClientTimeKey                               = pre(ClientRootKey) + "time"
	ClientTimeoutKey                            = pre(ClientRootKey) + "timeout"
	ClientMaxMessageSendSizeKey                 = pre(ClientRootKey) + "max_message_send_size"
	ClientMaxMessageReceiveSizeKey              = pre(ClientRootKey) + "max_message_receive_size"
	ClientMaxMessageSizeKey                     = pre(ClientRootKey) + "max_message_size"
	CollectorConfigPathKey                      = pre(CollectorRootKey) + "config_path"
	CollectorExportersKey                       = pre(CollectorRootKey) + "exporters"
	CollectorAttributeProcessorKey              = pre(CollectorProcessorsKey) + "attribute"
	CollectorDebugExporterKey                   = pre(CollectorExportersKey) + "debug"
	CollectorPrometheusExporterKey              = pre(CollectorExportersKey) + "prometheus_exporter"
	CollectorPrometheusExporterServerHostKey    = pre(CollectorPrometheusExporterKey) + "server_host"
	CollectorPrometheusExporterServerPortKey    = pre(CollectorPrometheusExporterKey) + "server_port"
	CollectorPrometheusExporterTLSKey           = pre(CollectorPrometheusExporterKey) + "tls"
	CollectorPrometheusExporterTLSCertKey       = pre(CollectorPrometheusExporterTLSKey) + "cert"
	CollectorPrometheusExporterTLSKeyKey        = pre(CollectorPrometheusExporterTLSKey) + "key"
	CollectorPrometheusExporterTLSCaKey         = pre(CollectorPrometheusExporterTLSKey) + "ca"
	CollectorPrometheusExporterTLSSkipVerifyKey = pre(CollectorPrometheusExporterTLSKey) + "skip_verify"
	CollectorPrometheusExporterTLSServerNameKey = pre(CollectorPrometheusExporterTLSKey) + "server_name"
	CollectorOtlpExportersKey                   = pre(CollectorExportersKey) + "otlp_exporters"
	CollectorProcessorsKey                      = pre(CollectorRootKey) + "processors"
	CollectorBatchProcessorKey                  = pre(CollectorProcessorsKey) + "batch"
	CollectorBatchProcessorSendBatchSizeKey     = pre(CollectorBatchProcessorKey) + "send_batch_size"
	CollectorBatchProcessorSendBatchMaxSizeKey  = pre(CollectorBatchProcessorKey) + "send_batch_max_size"
	CollectorBatchProcessorTimeoutKey           = pre(CollectorBatchProcessorKey) + "timeout"
	CollectorExtensionsKey                      = pre(CollectorRootKey) + "extensions"
	CollectorExtensionsHealthKey                = pre(CollectorExtensionsKey) + "health"
	CollectorExtensionsHealthServerHostKey      = pre(CollectorExtensionsHealthKey) + "server_host"
	CollectorExtensionsHealthServerPortKey      = pre(CollectorExtensionsHealthKey) + "server_port"
	CollectorExtensionsHealthPathKey            = pre(CollectorExtensionsHealthKey) + "path"
	CollectorExtensionsHealthTLSKey             = pre(CollectorExtensionsHealthKey) + "tls"
	CollectorExtensionsHealthTLSCaKey           = pre(CollectorExtensionsHealthTLSKey) + "ca"
	CollectorExtensionsHealthTLSCertKey         = pre(CollectorExtensionsHealthTLSKey) + "cert"
	CollectorExtensionsHealthTLSKeyKey          = pre(CollectorExtensionsHealthTLSKey) + "key"
	CollectorExtensionsHealthTLSServerNameKey   = pre(CollectorExtensionsHealthTLSKey) + "server_name"
	CollectorExtensionsHealthTLSSkipVerifyKey   = pre(CollectorExtensionsHealthTLSKey) + "skip_verify"
	CollectorExtensionsHeadersSetterKey         = pre(CollectorExtensionsKey) + "headers_setter"
	CollectorReceiversKey                       = pre(CollectorRootKey) + "receivers"
	CollectorLogKey                             = pre(CollectorRootKey) + "log"
	CollectorLogLevelKey                        = pre(CollectorLogKey) + "level"
	CollectorLogPathKey                         = pre(CollectorLogKey) + "path"
	CommandAuthKey                              = pre(CommandRootKey) + "auth"
	CommandAuthTokenKey                         = pre(CommandAuthKey) + "token"
	CommandAuthTokenPathKey                     = pre(CommandAuthKey) + "token-path"
	CommandServerHostKey                        = pre(CommandServerKey) + "host"
	CommandServerKey                            = pre(CommandRootKey) + "server"
	CommandServerPortKey                        = pre(CommandServerKey) + "port"
	CommandServerTypeKey                        = pre(CommandServerKey) + "type"
	CommandTLSKey                               = pre(CommandRootKey) + "tls"
	CommandTLSCaKey                             = pre(CommandTLSKey) + "ca"
	CommandTLSCertKey                           = pre(CommandTLSKey) + "cert"
	CommandTLSKeyKey                            = pre(CommandTLSKey) + "key"
	CommandTLSServerNameKey                     = pre(CommandTLSKey) + "server_name"
	CommandTLSSkipVerifyKey                     = pre(CommandTLSKey) + "skip_verify"
	LogLevelKey                                 = pre(LogLevelRootKey) + "level"
	LogPathKey                                  = pre(LogLevelRootKey) + "path"
	NginxReloadMonitoringPeriodKey              = pre(DataPlaneConfigRootKey, "nginx") + "reload_monitoring_period"
	NginxTreatWarningsAsErrorsKey               = pre(DataPlaneConfigRootKey, "nginx") + "treat_warnings_as_errors"
	NginxExcludeLogsKey                         = pre(DataPlaneConfigRootKey, "nginx") + "exclude_logs"
)

func pre(prefixes ...string) string {
	joined := strings.Join(prefixes, KeyDelimiter)
	return joined + KeyDelimiter
}
