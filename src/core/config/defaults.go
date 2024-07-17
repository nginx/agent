/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"os"
	"runtime"
	"time"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/client"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func ConfigFilePaths() []string {
	paths := []string{
		"/etc/nginx-manager/",
		"/etc/nginx-agent/",
		// Support for BSD style file hierarchy: https://www.freebsd.org/cgi/man.cgi?hier(7)
		// To keep them separate from the base system, user-installed applications are installed and configured under /usr/local/
		"/usr/local/etc/nginx-agent/",
	}

	path, err := os.Getwd()
	if err == nil {
		paths = append(paths, path)
	} else {
		log.Warn("unable to determine process's current directory")
	}

	return paths
}

var (
	Defaults = &Config{
		DynamicConfigPath: getDefaultDynamicConfPath(),
		CloudAccountID:    uuid.New().String(),
		Log: LogConfig{
			Level: "info",
			Path:  "/var/log/nginx-agent",
		},
		Server: Server{
			Command: "",
			Metrics: "",
			// token needs to be validated on the server side - can be overridden by the config value or the cli / environment variable
			// so setting to random uuid at the moment, tls connection won't work without the auth header
			Token: uuid.New().String(),
			Backoff: Backoff{
				InitialInterval:     client.DefaultBackoffSettings.InitialInterval,
				RandomizationFactor: client.DefaultBackoffSettings.Jitter,
				Multiplier:          client.DefaultBackoffSettings.Multiplier,
				MaxInterval:         client.DefaultBackoffSettings.MaxInterval,
				MaxElapsedTime:      client.DefaultBackoffSettings.MaxElapsedTime,
			},
		},
		Nginx: Nginx{
			Debug:                        false,
			NginxCountingSocket:          "unix:/var/run/nginx-agent/nginx.sock",
			NginxClientVersion:           7, // NGINX Plus R25+
			ConfigReloadMonitoringPeriod: 10 * time.Second,
			TreatWarningsAsErrors:        false,
		},
		ConfigDirs:            "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms",
		IgnoreDirectives:      []string{},
		AllowedDirectoriesMap: map[string]struct{}{},
		TLS: TLSConfig{
			Enable:     false,
			SkipVerify: false,
		},
		Dataplane: Dataplane{
			Status: Status{
				PollInterval:   30 * time.Second,
				ReportInterval: 24 * time.Hour,
			},
		},
		AgentMetrics: AgentMetrics{
			BulkSize:           20,
			ReportInterval:     1 * time.Minute,
			CollectionInterval: 15 * time.Second,
			Mode:               "aggregated",
			Backoff: Backoff{
				InitialInterval:     client.DefaultBackoffSettings.InitialInterval,
				RandomizationFactor: client.DefaultBackoffSettings.Jitter,
				Multiplier:          client.DefaultBackoffSettings.Multiplier,
				MaxInterval:         client.DefaultBackoffSettings.MaxInterval,
				MaxElapsedTime:      client.DefaultBackoffSettings.MaxElapsedTime,
			},
		},
		AgentAPI: AgentAPI{
			Host: "127.0.0.1",
		},
		Features:  agent_config.GetDefaultFeatures(),
		QueueSize: 100,
	}
	AllowedDirectoriesMap map[string]struct{}
)

const (
	MODULE = "NGINX-AGENT"

	DynamicConfigFileName           = "agent-dynamic.conf"
	DynamicConfigFileAbsPath        = "/var/lib/nginx-agent/agent-dynamic.conf"
	DynamicConfigFileAbsFreeBsdPath = "/var/db/nginx-agent/agent-dynamic.conf"
	ConfigFileName                  = "nginx-agent.conf"
	ConfigFileType                  = "yaml"
	LegacyEnvPrefix                 = "nms"
	EnvPrefix                       = "nginx_agent"
	ConfigPathKey                   = "path"
	DynamicConfigPathKey            = "dynamic_config_path"

	VersionKey          = "version"
	CloudAccountIdKey   = "cloudaccountid"
	LocationKey         = "location"
	DisplayNameKey      = "display_name"
	InstanceGroupKey    = "instance_group"
	ConfigDirsKey       = "config_dirs"
	TagsKey             = "tags"
	IgnoreDirectivesKey = "ignore_directives"
	QueueSizeKey        = "queue_size"

	// viper keys used in config
	LogKey = "log"

	LogLevel = LogKey + agent_config.KeyDelimiter + "level"
	LogPath  = LogKey + agent_config.KeyDelimiter + "path"

	// viper keys used in config
	ServerKey = "server"

	ServerHost     = ServerKey + agent_config.KeyDelimiter + "host"
	ServerGrpcPort = ServerKey + agent_config.KeyDelimiter + "grpcport"
	ServerToken    = ServerKey + agent_config.KeyDelimiter + "token"
	ServerMetrics  = ServerKey + agent_config.KeyDelimiter + "metrics"
	ServerCommand  = ServerKey + agent_config.KeyDelimiter + "command"
	ServerBackoff  = ServerKey + agent_config.KeyDelimiter + "backoff"

	// viper keys used in config
	BackoffKey                 = "backoff"
	BackoffInitialInterval     = ServerKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "initial_interval"
	BackoffRandomizationFactor = ServerKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "randomization_factor"
	BackoffMultiplier          = ServerKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "multiplier"
	BackoffMaxInterval         = ServerKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "max_interval"
	BackoffMaxElapsedTime      = ServerKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "max_elapsed_time"

	// viper keys used in config
	APIKey = "api"

	AgentAPIHost = APIKey + agent_config.KeyDelimiter + "host"
	AgentAPIPort = APIKey + agent_config.KeyDelimiter + "port"
	AgentAPICert = APIKey + agent_config.KeyDelimiter + "cert"
	AgentAPIKey  = APIKey + agent_config.KeyDelimiter + "key"

	// viper keys used in config
	TlsKey = "tls"

	TlsEnable     = TlsKey + agent_config.KeyDelimiter + "enable"
	TlsCert       = TlsKey + agent_config.KeyDelimiter + "cert"
	TlsPrivateKey = TlsKey + agent_config.KeyDelimiter + "key"
	TlsCa         = TlsKey + agent_config.KeyDelimiter + "ca"
	TlsSkipVerify = TlsKey + agent_config.KeyDelimiter + "skip_verify"

	// viper keys used in config
	NginxKey = "nginx"

	NginxExcludeLogs                  = NginxKey + agent_config.KeyDelimiter + "exclude_logs"
	NginxDebug                        = NginxKey + agent_config.KeyDelimiter + "debug"
	NginxCountingSocket               = NginxKey + agent_config.KeyDelimiter + "socket"
	NginxClientVersion                = NginxKey + agent_config.KeyDelimiter + "client_version"
	NginxConfigReloadMonitoringPeriod = NginxKey + agent_config.KeyDelimiter + "config_reload_monitoring_period"
	NginxTreatWarningsAsErrors        = NginxKey + agent_config.KeyDelimiter + "treat_warnings_as_errors"

	// viper keys used in config
	DataplaneKey = "dataplane"

	DataplaneEventsEnable         = DataplaneKey + agent_config.KeyDelimiter + "events_enable"
	DataplaneSyncEnable           = DataplaneKey + agent_config.KeyDelimiter + "sync_enable"
	DataplaneStatusPoll           = DataplaneKey + agent_config.KeyDelimiter + "status_poll_interval"
	DataplaneStatusReportInterval = DataplaneKey + agent_config.KeyDelimiter + "report_interval"

	// viper keys used in config
	MetricsKey = "metrics"

	MetricsBulkSize           = MetricsKey + agent_config.KeyDelimiter + "bulk_size"
	MetricsReportInterval     = MetricsKey + agent_config.KeyDelimiter + "report_interval"
	MetricsCollectionInterval = MetricsKey + agent_config.KeyDelimiter + "collection_interval"
	MetricsMode               = MetricsKey + agent_config.KeyDelimiter + "mode"
	MetricsBackoff            = MetricsKey + agent_config.KeyDelimiter + "backoff"
	// metrics backoff settings
	BackoffMetricsInitialInterval     = MetricsKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "initial_interval"
	BackoffMetricsRandomizationFactor = MetricsKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "randomization_factor"
	BackoffMetricsMultiplier          = MetricsKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "multiplier"
	BackoffMetricsMaxInterval         = MetricsKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "max_interval"
	BackoffMetricsMaxElapsedTime      = MetricsKey + agent_config.KeyDelimiter + BackoffKey + agent_config.KeyDelimiter + "max_elapsed_time"

	// DEPRECATED KEYS
	AdvancedMetricsKey                  = "advanced_metrics"
	AdvancedMetricsSocketPath           = AdvancedMetricsKey + agent_config.KeyDelimiter + "socket_path"
	AdvancedMetricsAggregationPeriod    = AdvancedMetricsKey + agent_config.KeyDelimiter + "aggregation_period"
	AdvancedMetricsPublishPeriod        = AdvancedMetricsKey + agent_config.KeyDelimiter + "publishing_period"
	AdvancedMetricsTableSizesLimits     = AdvancedMetricsKey + agent_config.KeyDelimiter + "table_sizes_limits"
	AdvancedMetricsTableSizesLimitsSTMS = AdvancedMetricsTableSizesLimits + agent_config.KeyDelimiter + "staging_table_max_size"
	AdvancedMetricsTableSizesLimitsSTT  = AdvancedMetricsTableSizesLimits + agent_config.KeyDelimiter + "staging_table_threshold"
	AdvancedMetricsTableSizesLimitsPTMS = AdvancedMetricsTableSizesLimits + agent_config.KeyDelimiter + "priority_table_max_size"
	AdvancedMetricsTableSizesLimitsPTT  = AdvancedMetricsTableSizesLimits + agent_config.KeyDelimiter + "priority_table_threshold"

	NginxAppProtectKey                    = "nginx_app_protect"
	NginxAppProtectReportInterval         = NginxAppProtectKey + agent_config.KeyDelimiter + "report_interval"
	NginxAppProtectPrecompiledPublication = NginxAppProtectKey + agent_config.KeyDelimiter + "precompiled_publication"

	NAPMonitoringKey                 = "nap_monitoring"
	NAPMonitoringCollectorBufferSize = NAPMonitoringKey + agent_config.KeyDelimiter + "collector_buffer_size"
	NAPMonitoringProcessorBufferSize = NAPMonitoringKey + agent_config.KeyDelimiter + "processor_buffer_size"
	NAPMonitoringSyslogIP            = NAPMonitoringKey + agent_config.KeyDelimiter + "syslog_ip"
	NAPMonitoringSyslogPort          = NAPMonitoringKey + agent_config.KeyDelimiter + "syslog_port"
	NAPMonitoringReportInterval      = NAPMonitoringKey + agent_config.KeyDelimiter + "report_interval"
	NAPMonitoringReportCount         = NAPMonitoringKey + agent_config.KeyDelimiter + "report_count"
)

var (
	agentFlags = []Registrable{
		&StringFlag{
			Name:         LogLevel,
			Usage:        "The desired verbosity level for logging messages from nginx-agent. Available options, in order of severity from highest to lowest, are: panic, fatal, error, info, debug, and trace.",
			DefaultValue: Defaults.Log.Level,
		},
		&StringFlag{
			Name:  LogPath,
			Usage: "The path to output log messages to. If the default path doesn't exist, log messages are output to stdout/stderr.",
		},
		&StringFlag{
			Name:  ServerHost,
			Usage: "The IP address of the server host. IPv4 addresses and hostnames are supported.",
		},
		&IntFlag{
			Name:  ServerGrpcPort,
			Usage: "The desired GRPC port to use for nginx-agent traffic.",
		},
		&StringFlag{
			Name:         ServerToken,
			Usage:        "An authentication token that grants nginx-agent access to the commander and metrics services. Auto-generated by default.",
			DefaultValue: Defaults.Server.Token,
		},
		&StringFlag{
			Name:         ServerMetrics,
			Usage:        "The name of the metrics server sent in the tls configuration.",
			DefaultValue: Defaults.Server.Metrics,
		},
		&StringFlag{
			Name:         ServerCommand,
			Usage:        "The name of the command server sent in the tls configuration.",
			DefaultValue: Defaults.Server.Command,
		},
		// API Config
		&StringFlag{
			Name:         AgentAPIHost,
			Usage:        "The host used by the Agent API.",
			DefaultValue: Defaults.AgentAPI.Host,
		},
		&IntFlag{
			Name:  AgentAPIPort,
			Usage: "The desired port to use for nginx-agent to expose for HTTP traffic.",
		},
		&StringFlag{
			Name:         AgentAPICert,
			Usage:        "The cert used by the Agent API.",
			DefaultValue: "",
		},
		&StringFlag{
			Name:         AgentAPIKey,
			Usage:        "The key used by the Agent API.",
			DefaultValue: "",
		},
		&StringFlag{
			Name:         DynamicConfigPathKey,
			Usage:        "Defines the path of the Agent dynamic config file.",
			DefaultValue: getDefaultDynamicConfPath(),
		},
		&StringFlag{
			Name:         ConfigDirsKey,
			Usage:        "Defines the paths that you want to grant nginx-agent read/write access to. This key is formatted as a string and follows Unix PATH format.",
			DefaultValue: Defaults.ConfigDirs,
		},
		&StringSliceFlag{
			Name:  TagsKey,
			Usage: "A comma-separated list of tags to add to the current instance or machine, to be used for inventory purposes.",
		},
		&StringSliceFlag{
			Name:         agent_config.FeaturesKey,
			Usage:        "A comma-separated list of features enabled for the agent.",
			DefaultValue: agent_config.GetDefaultFeatures(),
		},
		// NGINX Config
		&StringSliceFlag{
			Name:         IgnoreDirectivesKey,
			Usage:        "A comma-separated list of ignoring directives which contain sensitive info.",
			DefaultValue: Defaults.IgnoreDirectives,
		},
		&StringFlag{
			Name:  NginxExcludeLogs,
			Usage: "One or more NGINX access log paths that you want to exclude from metrics collection. This key is formatted as a string and multiple values should be provided as a comma-separated list.",
		},
		&StringFlag{
			Name:         NginxCountingSocket,
			Usage:        "The NGINX Plus counting unix socket location.",
			DefaultValue: Defaults.Nginx.NginxCountingSocket,
		},
		&DurationFlag{
			Name:         NginxConfigReloadMonitoringPeriod,
			Usage:        "The duration the NGINX Agent will monitor error logs after a NGINX reload",
			DefaultValue: Defaults.Nginx.ConfigReloadMonitoringPeriod,
		},
		&BoolFlag{
			Name:         NginxTreatWarningsAsErrors,
			Usage:        "On nginx -t, treat warnings as failures on configuration application.",
			DefaultValue: Defaults.Nginx.TreatWarningsAsErrors,
		},
		// Metrics
		&DurationFlag{
			Name:         MetricsCollectionInterval,
			Usage:        "Sets the interval, in seconds, at which metrics are collected.",
			DefaultValue: Defaults.AgentMetrics.CollectionInterval,
		},
		&StringFlag{
			Name:         MetricsMode,
			Usage:        "Sets the desired metrics collection mode: streaming or aggregation.",
			DefaultValue: Defaults.AgentMetrics.Mode,
		},
		&IntFlag{
			Name:         MetricsBulkSize,
			Usage:        "The amount of metrics reports collected before sending the data back to the server.",
			DefaultValue: Defaults.AgentMetrics.BulkSize,
		},
		&DurationFlag{
			Name:         MetricsReportInterval,
			Usage:        "The polling period specified for a single set of metrics being collected.",
			DefaultValue: Defaults.AgentMetrics.ReportInterval,
		},
		// TLS Config
		&BoolFlag{
			Name:         TlsEnable,
			Usage:        "Enables TLS for secure communications.",
			DefaultValue: Defaults.TLS.Enable,
		},
		&StringFlag{
			Name:  TlsCert,
			Usage: "The path to the certificate file to use for TLS.",
		},
		&StringFlag{
			Name:  TlsPrivateKey,
			Usage: "The path to the certificate key file to use for TLS.",
		},
		&StringFlag{
			Name:  TlsCa,
			Usage: "The path to the CA certificate file to use for TLS.",
		},
		&BoolFlag{
			Name:         TlsSkipVerify,
			Usage:        "Only intended for demonstration, sets InsecureSkipVerify for gRPC TLS credentials",
			DefaultValue: Defaults.TLS.SkipVerify,
		},
		// Dataplane
		&DurationFlag{
			Name:         DataplaneStatusPoll,
			Usage:        "The frequency the agent will check the dataplane for changes. Used as a \"heartbeat\" to keep the gRPC connections alive.",
			DefaultValue: Defaults.Dataplane.Status.PollInterval,
		},
		&DurationFlag{
			Name:         DataplaneStatusReportInterval,
			Usage:        "The amount of time the agent will report on the dataplane. After this period of time it will send a snapshot of the dataplane information.",
			DefaultValue: Defaults.Dataplane.Status.ReportInterval,
		},
		// Other Config
		&StringFlag{
			Name:  DisplayNameKey,
			Usage: "The instance's 'name' value.",
		},
		&StringFlag{
			Name:  InstanceGroupKey,
			Usage: "The instance's 'group' value.",
		},
		&IntFlag{
			Name:  QueueSizeKey,
			Usage: "The size of the NGINX Agent internal queue.",
		},
	}
	deprecatedFlags = []Registrable{
		// Advanced Metrics
		&StringFlag{
			Name:  AdvancedMetricsSocketPath,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&DurationFlag{
			Name:  AdvancedMetricsAggregationPeriod,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&DurationFlag{
			Name:  AdvancedMetricsPublishPeriod,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  AdvancedMetricsTableSizesLimitsPTMS,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  AdvancedMetricsTableSizesLimitsPTT,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  AdvancedMetricsTableSizesLimitsSTMS,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  AdvancedMetricsTableSizesLimitsSTT,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		// Nginx App Protect
		&DurationFlag{
			Name:  NginxAppProtectReportInterval,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&BoolFlag{
			Name:  NginxAppProtectPrecompiledPublication,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		// NAP Monitoring
		&IntFlag{
			Name:  NAPMonitoringCollectorBufferSize,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  NAPMonitoringProcessorBufferSize,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  NAPMonitoringSyslogIP,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&IntFlag{
			Name:  NAPMonitoringSyslogPort,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
	}
)

func getDefaultDynamicConfPath() string {
	if runtime.GOOS == "freebsd" {
		return DynamicConfigFileAbsFreeBsdPath
	} else {
		return DynamicConfigFileAbsPath
	}
}
