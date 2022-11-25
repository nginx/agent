package config

import (
	"time"

	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
)

type Config struct {
	Path                  string              `yaml:"-"`
	DynamicConfigPath     string              `yaml:"-"`
	ClientID              string              `mapstructure:"agent_id" yaml:"-"`
	CloudAccountID        string              `mapstructure:"cloud_account" yaml:"-"`
	Server                Server              `mapstructure:"server" yaml:"-"`
	AgentAPI              AgentAPI            `mapstructure:"api" yaml:"-"`
	ConfigDirs            string              `mapstructure:"config-dirs" yaml:"-"`
	Log                   LogConfig           `mapstructure:"log" yaml:"-"`
	TLS                   TLSConfig           `mapstructure:"tls" yaml:"-"`
	Nginx                 Nginx               `mapstructure:"nginx" yaml:"-"`
	Dataplane             Dataplane           `mapstructure:"dataplane" yaml:"-"`
	AgentMetrics          AgentMetrics        `mapstructure:"metrics" yaml:"-"`
	Tags                  []string            `mapstructure:"tags" yaml:"tags,omitempty"`
	Features              []string            `mapstructure:"features" yaml:"features,omitempty"`
	Extensions            []string            `mapstructure:"extensions" yaml:"extensions,omitempty"`
	Updated               time.Time           `yaml:"-"` // update time of the config file
	AllowedDirectoriesMap map[string]struct{} `yaml:"-"`
	DisplayName           string              `mapstructure:"display_name" yaml:"display_name,omitempty"`
	InstanceGroup         string              `mapstructure:"instance_group" yaml:"instance_group,omitempty"`
	AdvancedMetrics       AdvancedMetrics     `mapstructure:"advanced_metrics" yaml:"advanced_metrics,omitempty"`
	NginxAppProtect       NginxAppProtect     `mapstructure:"nginx_app_protect" yaml:"nginx_app_protect,omitempty"`
	NAPMonitoring         NAPMonitoring       `mapstructure:"nap_monitoring" yaml:"nap_monitoring,omitempty"`
}

type Server struct {
	Host     string `mapstructure:"host" yaml:"-"`
	GrpcPort int    `mapstructure:"grpcPort" yaml:"-"`
	Token    string `mapstructure:"token" yaml:"-"`
	Metrics  string `mapstructure:"metrics" yaml:"-"`
	Command  string `mapstructure:"command" yaml:"-"`
	// This is internal and shouldnt be exposed as a flag
	Target string `mapstructure:"target" yaml:"-"`
}

type AgentAPI struct {
	Port int    `mapstructure:"port" yaml:"-"`
	Cert string `mapstructure:"cert" yaml:"-"`
	Key  string `mapstructure:"key" yaml:"-"`
}

// LogConfig for logging
type LogConfig struct {
	Level string `mapstructure:"level" yaml:"-"`
	Path  string `mapstructure:"path" yaml:"-"`
}

// TLSConfig for securing communications
type TLSConfig struct {
	Enable     bool   `mapstructure:"enable" yaml:"-"`
	Cert       string `mapstructure:"cert" yaml:"-"`
	Key        string `mapstructure:"key" yaml:"-"`
	Ca         string `mapstructure:"ca" yaml:"-"`
	SkipVerify bool   `mapstructure:"skip_verify" yaml:"-"`
}

// Nginx settings
type Nginx struct {
	ExcludeLogs         string `mapstructure:"exclude_logs" yaml:"-"`
	Debug               bool   `mapstructure:"debug" yaml:"-"`
	NginxCountingSocket string `mapstructure:"socket" yaml:"-"`
	NginxClientVersion  int    `mapstructure:"client_version" yaml:"-"`
}

type Dataplane struct {
	Status Status `mapstructure:"status" yaml:"-"`
}

// Status polling for heartbeat settings
type Status struct {
	PollInterval   time.Duration `mapstructure:"poll_interval" yaml:"-"`
	ReportInterval time.Duration `mapstructure:"report_interval" yaml:"-"`
}

// AgentMetrics for system/agent metrics
type AgentMetrics struct {
	BulkSize           int           `mapstructure:"bulk_size" yaml:"-"`
	ReportInterval     time.Duration `mapstructure:"report_interval" yaml:"-"`
	CollectionInterval time.Duration `mapstructure:"collection_interval" yaml:"-"`
	Mode               string        `mapstructure:"mode" yaml:"-"`
}

type AdvancedMetrics struct {
	SocketPath        string                            `mapstructure:"socket_path" yaml:"-"`
	AggregationPeriod time.Duration                     `mapstructure:"aggregation_period" yaml:"-"`
	PublishingPeriod  time.Duration                     `mapstructure:"publishing_period" yaml:"-"`
	TableSizesLimits  advanced_metrics.TableSizesLimits `mapstructure:"table_sizes_limits" yaml:"-"`
}

type NginxAppProtect struct {
	ReportInterval time.Duration `mapstructure:"report_interval" yaml:"-"`
}

type NAPMonitoring struct {
	CollectorBufferSize int           `mapstructure:"collector_buffer_size" yaml:"-"`
	ProcessorBufferSize int           `mapstructure:"processor_buffer_size" yaml:"-"`
	SyslogIP            string        `mapstructure:"syslog_ip" yaml:"-"`
	SyslogPort          int           `mapstructure:"syslog_port" yaml:"-"`
	ReportInterval      time.Duration `mapstructure:"report_interval" yaml:"-"`
	ReportCount         int           `mapstructure:"report_count" yaml:"-"`
}
