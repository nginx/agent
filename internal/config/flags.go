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
	LogLevelRootKey                             = "log"
	CollectorRootKey                            = "collector"
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
	ClientMaxMessageSendSizeKey    = pre(ClientRootKey) + "max_message_send_size"
	ClientMaxMessageRecieveSizeKey = pre(ClientRootKey) + "max_message_receive_size"
	ClientMaxMessageSizeKey        = pre(ClientRootKey) + "max_message_size"
	CollectorConfigPathKey         = pre(CollectorRootKey) + "config_path"
	CollectorExportersKey          = pre(CollectorRootKey) + "exporters"
	CollectorProcessorsKey         = pre(CollectorRootKey) + "processors"
	CollectorHealthKey             = pre(CollectorRootKey) + "health"
	CollectorReceiversKey          = pre(CollectorRootKey) + "receivers"
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
	LogLevelKey                    = pre(LogLevelRootKey) + "level"
	LogPathKey                     = pre(LogLevelRootKey) + "path"
	NginxReloadMonitoringPeriodKey = pre(DataPlaneConfigRootKey, "nginx") + "reload_monitoring_period"
	NginxTreatWarningsAsErrorsKey  = pre(DataPlaneConfigRootKey, "nginx") + "treat_warnings_as_error"
	OTLPExportURLKey               = pre(CollectorRootKey) + "otlp_export_url"
	OTLPReceiverURLKey             = pre(CollectorRootKey) + "otlp_receiver_url"
)

func pre(prefixes ...string) string {
	joined := strings.Join(prefixes, KeyDelimiter)
	return joined + KeyDelimiter
}
