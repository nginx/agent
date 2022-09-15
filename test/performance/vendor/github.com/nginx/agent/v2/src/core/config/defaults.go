package config

import (
	"os"
	"time"

	"github.com/google/uuid"
	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
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
		CloudAccountID: uuid.New().String(),
		Log: LogConfig{
			Level: "info",
			Path:  "/var/log/nginx-agent",
		},
		Server: Server{
			Host:     "127.0.0.1",
			GrpcPort: 443,
			Command:  "",
			Metrics:  "",
			// token needs to be validated on the server side - can be overridden by the config value or the cli / environment variable
			// so setting to random uuid at the moment, tls connection won't work without the auth header
			Token: uuid.New().String(),
		},
		Nginx: Nginx{
			Debug:               false,
			NginxCountingSocket: "unix:/var/run/nginx-agent/nginx.sock",
		},
		ConfigDirs:            "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms",
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
			Mode:               "aggregation",
		},
		AdvancedMetrics: AdvancedMetrics{
			SocketPath:        "/var/run/nginx-agent/advanced-metrics.sock",
			AggregationPeriod: time.Second * 10,
			PublishingPeriod:  time.Second * 30,
			TableSizesLimits: advanced_metrics.TableSizesLimits{
				StagingTableThreshold:  1000,
				StagingTableMaxSize:    1000,
				PriorityTableThreshold: 1000,
				PriorityTableMaxSize:   1000,
			},
		},
		Features: []string{
			FeatureRegistration,
			FeatureNginxConfig,
			FeatureNginxSSLConfig,
			FeatureNginxCounting,
			FeatureMetrics,
			FeatureMetricsThrottle,
			FeatureDataPlaneStatus,
			FeatureProcessWatcher,
			FeatureFileWatcher,
			FeatureActivityEvents,
		},
	}
	AllowedDirectoriesMap map[string]struct{}
)

const (
	DynamicConfigFileName    = "agent-dynamic.conf"
	DynamicConfigFileAbsPath = "/etc/nginx-agent/agent-dynamic.conf"
	ConfigFileName           = "nginx-agent.conf"
	ConfigFileType           = "yaml"
	KeyDelimiter             = "_"
	EnvPrefix                = "nms"
	ConfigPathKey            = "path"
	DynamicConfigPathKey     = "dynamic-config-path"

	CloudAccountIdKey = "cloudaccountid"
	LocationKey       = "location"
	DisplayNameKey    = "display_name"
	InstanceGroupKey  = "instance_group"
	ConfigDirsKey     = "config_dirs"
	TagsKey           = "tags"

	// viper keys used in config
	LogKey = "log"

	LogLevel = LogKey + KeyDelimiter + "level"
	LogPath  = LogKey + KeyDelimiter + "path"

	// viper keys used in config
	ServerKey = "server"

	ServerHost     = ServerKey + KeyDelimiter + "host"
	ServerGrpcport = ServerKey + KeyDelimiter + "grpcport"
	ServerToken    = ServerKey + KeyDelimiter + "token"
	ServerMetrics  = ServerKey + KeyDelimiter + "metrics"
	ServerCommand  = ServerKey + KeyDelimiter + "command"

	// viper keys used in config
	TlsKey = "tls"

	TlsEnable     = TlsKey + KeyDelimiter + "enable"
	TlsCert       = TlsKey + KeyDelimiter + "cert"
	TlsPrivateKey = TlsKey + KeyDelimiter + "key"
	TlsCa         = TlsKey + KeyDelimiter + "ca"
	TlsSkipVerify = TlsKey + KeyDelimiter + "skip_verify"

	// viper keys used in config
	NginxKey = "nginx"

	NginxExcludeLogs    = NginxKey + KeyDelimiter + "exclude_logs"
	NginxDebug          = NginxKey + KeyDelimiter + "debug"
	NginxCountingSocket = NginxKey + KeyDelimiter + "socket"

	// viper keys used in config
	DataplaneKey = "dataplane"

	DataplaneEventsEnable         = DataplaneKey + KeyDelimiter + "events_enable"
	DataplaneSyncEnable           = DataplaneKey + KeyDelimiter + "sync_enable"
	DataplaneStatusPoll           = DataplaneKey + KeyDelimiter + "status_poll_interval"
	DataplaneStatusReportInterval = DataplaneKey + KeyDelimiter + "report_interval"

	// viper keys used in config
	MetricsKey = "metrics"

	MetricsBulkSize           = MetricsKey + KeyDelimiter + "bulk_size"
	MetricsReportInterval     = MetricsKey + KeyDelimiter + "report_interval"
	MetricsCollectionInterval = MetricsKey + KeyDelimiter + "collection_interval"
	MetricsMode               = MetricsKey + KeyDelimiter + "mode"

	// viper keys used in config
	AdvancedMetricsKey = "advanced_metrics"

	AdvancedMetricsSocketPath           = AdvancedMetricsKey + KeyDelimiter + "socket_path"
	AdvancedMetricsAggregationPeriod    = AdvancedMetricsKey + KeyDelimiter + "aggregation_period"
	AdvancedMetricsPublishPeriod        = AdvancedMetricsKey + KeyDelimiter + "publishing_period"
	AdvancedMetricsTableSizesLimits     = AdvancedMetricsKey + KeyDelimiter + "table_sizes_limits"
	AdvancedMetricsTableSizesLimitsSTMS = AdvancedMetricsTableSizesLimits + KeyDelimiter + "staging_table_max_size"
	AdvancedMetricsTableSizesLimitsSTT  = AdvancedMetricsTableSizesLimits + KeyDelimiter + "staging_table_threshold"
	AdvancedMetricsTableSizesLimitsPTMS = AdvancedMetricsTableSizesLimits + KeyDelimiter + "priority_table_max_size"
	AdvancedMetricsTableSizesLimitsPTT  = AdvancedMetricsTableSizesLimits + KeyDelimiter + "priority_table_threshold"

	// viper keys used in config
	NginxAppProtectKey = "nginx_app_protect"

	NginxAppProtectReportInterval = NginxAppProtectKey + KeyDelimiter + "report_interval"

	// viper keys used in config
	FeaturesKey = "features"

	FeatureRegistration    = FeaturesKey + KeyDelimiter + "registration"
	FeatureNginxConfig     = FeaturesKey + KeyDelimiter + "nginx-config"
	FeatureNginxSSLConfig  = FeaturesKey + KeyDelimiter + "nginx-ssl-config"
	FeatureNginxCounting   = FeaturesKey + KeyDelimiter + "nginx-counting"
	FeatureMetrics         = FeaturesKey + KeyDelimiter + "metrics"
	FeatureMetricsThrottle = FeaturesKey + KeyDelimiter + "metrics-throttle"
	FeatureDataPlaneStatus = FeaturesKey + KeyDelimiter + "dataplane-status"
	FeatureProcessWatcher  = FeaturesKey + KeyDelimiter + "process-watcher"
	FeatureFileWatcher     = FeaturesKey + KeyDelimiter + "file-watcher"
	FeatureActivityEvents  = FeaturesKey + KeyDelimiter + "activity-events"

	// DEPRECATED KEYS
	NginxBinPathKey       = "nginx_bin_path"
	NginxPIDPathKey       = "nginx_pid_path"
	NginxStubStatusURLKey = "nginx_stub_status"
	NginxPlusAPIURLKey    = "nginx_plus_api"
	NginxMetricsPollKey   = "nginx_metrics_poll_interval"

	MetricsEnableTLSKey   = "metrics_tls_enable"
	MetricsTLSCertPathKey = "metrics_tls_cert"
	MetricsTLSKeyPathKey  = "metrics_tls_key"
	MetricsTLSCAPathKey   = "metrics_tls_ca"
)

var (
	agentFlags = []Registrable{
		&StringFlag{
			Name:         LogLevel,
			Usage:        "The desired verbosity level for logging messages from nginx-agent. Available options, in order of severity from highest to lowest, are: panic, fatal, error, info, debug, and trace.",
			DefaultValue: Defaults.Log.Level,
		},
		&StringFlag{
			Name:         LogPath,
			Usage:        "The path to output log messages to. If the default path doesn't exist, log messages are output to stdout/stderr.",
			DefaultValue: Defaults.Log.Path,
		},
		&StringFlag{
			Name:         ServerHost,
			Usage:        "The IP address of the server host. IPv4 addresses and hostnames are supported.",
			DefaultValue: Defaults.Server.Host,
		},
		&IntFlag{
			Name:         ServerGrpcport,
			Usage:        "The desired GRPC port to use for nginx-agent traffic.",
			DefaultValue: Defaults.Server.GrpcPort,
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
			Name:  FeaturesKey,
			Usage: "A comma-separated list of features enabled for the agent.",
			DefaultValue: Defaults.Features,
		},
		// NGINX Config
		&StringFlag{
			Name:  NginxExcludeLogs,
			Usage: "One or more NGINX access log paths that you want to exclude from metrics collection. This key is formatted as a string and multiple values should be provided as a comma-separated list.",
		},
		&StringFlag{
			Name:         NginxCountingSocket,
			Usage:        "The NGINX Plus counting unix socket location.",
			DefaultValue: Defaults.Nginx.NginxCountingSocket,
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
		// Advanced Metrics
		&StringFlag{
			Name:         AdvancedMetricsSocketPath,
			Usage:        "The advanced metrics socket location.",
			DefaultValue: Defaults.AdvancedMetrics.SocketPath,
		},
		// change to advanced metrics collection interval
		&DurationFlag{
			Name:         AdvancedMetricsAggregationPeriod,
			Usage:        "Sets the interval, in seconds, at which advanced metrics are collected.",
			DefaultValue: Defaults.AdvancedMetrics.AggregationPeriod,
		},
		// change to advanced metrics report interval
		&DurationFlag{
			Name:         AdvancedMetricsPublishPeriod,
			Usage:        "The polling period specified for a single set of advanced metrics being collected.",
			DefaultValue: Defaults.AdvancedMetrics.PublishingPeriod,
		},
		&IntFlag{
			Name:         AdvancedMetricsTableSizesLimitsPTMS,
			Usage:        "Default Maximum Size of the Priority Table.",
			DefaultValue: Defaults.AdvancedMetrics.TableSizesLimits.PriorityTableMaxSize,
		},
		&IntFlag{
			Name:         AdvancedMetricsTableSizesLimitsPTT,
			Usage:        "Default Threshold of the Priority Table - normally a value which is a percentage of the corresponding Default Maximum Size of the Priority Table (<100%, but its value is not an actual percentage, i.e 88%, rather 88%*AdvancedMetricsTableSizesLimitsPTMS).",
			DefaultValue: Defaults.AdvancedMetrics.TableSizesLimits.PriorityTableThreshold,
		},
		&IntFlag{
			Name:         AdvancedMetricsTableSizesLimitsSTMS,
			Usage:        "Default Maximum Size of the Staging Table.",
			DefaultValue: Defaults.AdvancedMetrics.TableSizesLimits.StagingTableMaxSize,
		},
		&IntFlag{
			Name:         AdvancedMetricsTableSizesLimitsSTT,
			Usage:        "AdvancedMetricsTableSizesLimitsSTT - Default Threshold of the Staging Table - normally a value which is a percentage of the corresponding Default Maximum Size of the Staging Table (<100%, but its value is not an actual percentage, i.e 88%, rather 88%*AdvancedMetricsTableSizesLimitsSTMS).",
			DefaultValue: Defaults.AdvancedMetrics.TableSizesLimits.StagingTableThreshold,
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
		// Nginx App Protect
		&DurationFlag{
			Name:  NginxAppProtectReportInterval,
			Usage: "The period of time the agent will check for App Protect software changes on the dataplane",
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
	}
	deprecatedFlags = []Registrable{
		&StringFlag{
			Name:  "metadata",
			Usage: "DEPRECATED; use --server-host instead.",
		},
		&StringFlag{
			Name:  ServerKey,
			Usage: "DEPRECATED; use --server-grpcport instead.",
		},
		&StringFlag{
			Name:  "metrics_server",
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  "api_token",
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  DataplaneSyncEnable,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  DataplaneEventsEnable,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  LocationKey,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		// NGINX Config
		&StringFlag{
			Name:  NginxBinPathKey,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  NginxPIDPathKey,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  NginxStubStatusURLKey,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&StringFlag{
			Name:  NginxPlusAPIURLKey,
			Usage: "DEPRECATED; no replacement due to change in functionality.",
		},
		&DurationFlag{
			Name:  NginxMetricsPollKey,
			Usage: "DEPRECATED; use --metrics-collection-interval instead.",
		},
		// Metrics TLS Config
		&BoolFlag{
			Name:  MetricsEnableTLSKey,
			Usage: "DEPRECATED; use --tls-enable instead.",
		},
		&StringFlag{
			Name:  MetricsTLSCertPathKey,
			Usage: "DEPRECATED; use --tls-cert instead.",
		},
		&StringFlag{
			Name:  MetricsTLSKeyPathKey,
			Usage: "DEPRECATED; use --tls-key instead.",
		},
		&StringFlag{
			Name:  MetricsTLSCAPathKey,
			Usage: "DEPRECATED; use --tls-ca instead.",
		},
	}
)
