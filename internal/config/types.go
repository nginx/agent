// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ServerType string

const (
	Grpc ServerType = "grpc"
)

var serverTypes = map[string]ServerType{
	"grpc": Grpc,
}

func parseServerType(str string) (ServerType, bool) {
	c, ok := serverTypes[strings.ToLower(str)]
	return c, ok
}

type (
	Config struct {
		Command            *Command         `yaml:"command"             mapstructure:"command"`
		Log                *Log             `yaml:"log"                 mapstructure:"log"`
		DataPlaneConfig    *DataPlaneConfig `yaml:"data_plane_config"   mapstructure:"data_plane_config"`
		Client             *Client          `yaml:"client"              mapstructure:"client"`
		Collector          *Collector       `yaml:"collector"           mapstructure:"collector"`
		Watchers           *Watchers        `yaml:"watchers"            mapstructure:"watchers"`
		Labels             map[string]any   `yaml:"labels"              mapstructure:"labels"`
		Version            string           `yaml:"-"`
		Path               string           `yaml:"-"`
		UUID               string           `yaml:"-"`
		AllowedDirectories []string         `yaml:"allowed_directories" mapstructure:"allowed_directories"`
		Features           []string         `yaml:"features"            mapstructure:"features"`
	}

	Log struct {
		Level string `yaml:"level" mapstructure:"level"`
		Path  string `yaml:"path"  mapstructure:"path"`
	}

	DataPlaneConfig struct {
		Nginx *NginxDataPlaneConfig `yaml:"nginx" mapstructure:"nginx"`
	}

	NginxDataPlaneConfig struct {
		ExcludeLogs            []string      `yaml:"exclude_logs"             mapstructure:"exclude_logs"`
		ReloadMonitoringPeriod time.Duration `yaml:"reload_monitoring_period" mapstructure:"reload_monitoring_period"`
		TreatWarningsAsErrors  bool          `yaml:"treat_warnings_as_errors" mapstructure:"treat_warnings_as_errors"`
	}

	Client struct {
		HTTP    *HTTP    `yaml:"http"    mapstructure:"http"`
		Grpc    *GRPC    `yaml:"grpc"    mapstructure:"grpc"`
		Backoff *BackOff `yaml:"backoff" mapstructure:"backoff"`
	}

	HTTP struct {
		Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`
	}

	BackOff struct {
		InitialInterval     time.Duration `yaml:"initial_interval"     mapstructure:"initial_interval"`
		MaxInterval         time.Duration `yaml:"max_interval"         mapstructure:"max_interval"`
		MaxElapsedTime      time.Duration `yaml:"max_elapsed_time"     mapstructure:"max_elapsed_time"`
		RandomizationFactor float64       `yaml:"randomization_factor" mapstructure:"randomization_factor"`
		Multiplier          float64       `yaml:"multiplier"           mapstructure:"multiplier"`
	}

	GRPC struct {
		KeepAlive *KeepAlive `yaml:"keepalive" mapstructure:"keepalive"`
		// if MaxMessageSize is size set then we use that value,
		// otherwise MaxMessageRecieveSize and MaxMessageSendSize for individual settings
		MaxMessageSize        int `yaml:"max_message_size"         mapstructure:"max_message_size"`
		MaxMessageReceiveSize int `yaml:"max_message_receive_size" mapstructure:"max_message_receive_size"`
		MaxMessageSendSize    int `yaml:"max_message_send_size"    mapstructure:"max_message_send_size"`
	}

	KeepAlive struct {
		Timeout             time.Duration `yaml:"timeout"               mapstructure:"timeout"`
		Time                time.Duration `yaml:"time"                  mapstructure:"time"`
		PermitWithoutStream bool          `yaml:"permit_without_stream" mapstructure:"permit_without_stream"`
	}

	Collector struct {
		ConfigPath string     `yaml:"config_path" mapstructure:"config_path"`
		Log        *Log       `yaml:"log"         mapstructure:"log"`
		Exporters  Exporters  `yaml:"exporters"   mapstructure:"exporters"`
		Extensions Extensions `yaml:"extensions"  mapstructure:"extensions"`
		Processors Processors `yaml:"processors"  mapstructure:"processors"`
		Receivers  Receivers  `yaml:"receivers"   mapstructure:"receivers"`
	}

	Exporters struct {
		Debug              *DebugExporter      `yaml:"debug"               mapstructure:"debug"`
		PrometheusExporter *PrometheusExporter `yaml:"prometheus_exporter" mapstructure:"prometheus_exporter"`
		OtlpExporters      []OtlpExporter      `yaml:"otlp_exporters"      mapstructure:"otlp_exporters"`
	}

	OtlpExporter struct {
		Server        *ServerConfig `yaml:"server"        mapstructure:"server"`
		TLS           *TLSConfig    `yaml:"tls"           mapstructure:"tls"`
		Compression   string        `yaml:"compression"   mapstructure:"compression"`
		Authenticator string        `yaml:"authenticator" mapstructure:"authenticator"`
	}

	Extensions struct {
		Health        *Health        `yaml:"health"         mapstructure:"health"`
		HeadersSetter *HeadersSetter `yaml:"headers_setter" mapstructure:"headers_setter"`
	}

	Health struct {
		Server *ServerConfig `yaml:"server" mapstructure:"server"`
		TLS    *TLSConfig    `yaml:"tls"    mapstructure:"tls"`
		Path   string        `yaml:"path"   mapstructure:"path"`
	}

	HeadersSetter struct {
		Headers []Header `yaml:"headers" mapstructure:"headers"`
	}

	Header struct {
		Action       string `yaml:"action"        mapstructure:"action"`
		Key          string `yaml:"key"           mapstructure:"key"`
		Value        string `yaml:"value"         mapstructure:"value"`
		DefaultValue string `yaml:"default_value" mapstructure:"default_value"`
		FromContext  string `yaml:"from_context"  mapstructure:"from_context"`
	}

	DebugExporter struct{}

	PrometheusExporter struct {
		Server *ServerConfig `yaml:"server" mapstructure:"server"`
		TLS    *TLSConfig    `yaml:"tls"    mapstructure:"tls"`
	}

	// OTel Collector Processors configuration.
	Processors struct {
		Attribute *Attribute `yaml:"attribute" mapstructure:"attribute"`
		Resource  *Resource  `yaml:"resource"  mapstructure:"resource"`
		Batch     *Batch     `yaml:"batch"     mapstructure:"batch"`
	}

	Attribute struct {
		Actions []Action `yaml:"actions" mapstructure:"actions"`
	}

	Action struct {
		Key    string `yaml:"key"    mapstructure:"key"`
		Action string `yaml:"action" mapstructure:"action"`
		Value  string `yaml:"value"  mapstructure:"value"`
	}

	Resource struct {
		Attributes []ResourceAttribute `yaml:"attributes" mapstructure:"attributes"`
	}

	ResourceAttribute struct {
		Key    string `yaml:"key"    mapstructure:"key"`
		Action string `yaml:"action" mapstructure:"action"`
		Value  string `yaml:"value"  mapstructure:"value"`
	}

	Batch struct {
		SendBatchSize    uint32        `yaml:"send_batch_size"     mapstructure:"send_batch_size"`
		SendBatchMaxSize uint32        `yaml:"send_batch_max_size" mapstructure:"send_batch_max_size"`
		Timeout          time.Duration `yaml:"timeout"             mapstructure:"timeout"`
	}

	// OTel Collector Receiver configuration.
	Receivers struct {
		ContainerMetrics   *ContainerMetricsReceiver `yaml:"container_metrics"    mapstructure:"container_metrics"`
		HostMetrics        *HostMetrics              `yaml:"host_metrics"         mapstructure:"host_metrics"`
		OtlpReceivers      []OtlpReceiver            `yaml:"otlp_receivers"       mapstructure:"otlp_receivers"`
		NginxReceivers     []NginxReceiver           `yaml:"nginx_receivers"      mapstructure:"nginx_receivers"`
		NginxPlusReceivers []NginxPlusReceiver       `yaml:"nginx_plus_receivers" mapstructure:"nginx_plus_receivers"`
		TcplogReceivers    []TcplogReceiver          `yaml:"tcplog_receivers"     mapstructure:"tcplog_receivers"`
	}

	OtlpReceiver struct {
		Server        *ServerConfig  `yaml:"server" mapstructure:"server"`
		Auth          *AuthConfig    `yaml:"auth"   mapstructure:"auth"`
		OtlpTLSConfig *OtlpTLSConfig `yaml:"tls"    mapstructure:"tls"`
	}

	TcplogReceiver struct {
		ListenAddress string     `yaml:"listen_address" mapstructure:"listen_address"`
		Operators     []Operator `yaml:"operators"      mapstructure:"operators"`
	}

	// There are many types of operators with different field names so we use a generic map to store the fields.
	// See here for more info:
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/stanza/docs/operators/README.md
	Operator struct {
		Fields map[string]string `yaml:"fields" mapstructure:"fields"`
		Type   string            `yaml:"type"   mapstructure:"type"`
	}

	NginxReceiver struct {
		InstanceID string      `yaml:"instance_id" mapstructure:"instance_id"`
		StubStatus APIDetails  `yaml:"api_details" mapstructure:"api_details"`
		AccessLogs []AccessLog `yaml:"access_logs" mapstructure:"access_logs"`
	}

	APIDetails struct {
		URL      string `yaml:"url"      mapstructure:"url"`
		Listen   string `yaml:"listen"   mapstructure:"listen"`
		Location string `yaml:"location" mapstructure:"location"`
	}

	AccessLog struct {
		FilePath  string `yaml:"file_path"  mapstructure:"file_path"`
		LogFormat string `yaml:"log_format" mapstructure:"log_format"`
	}

	NginxPlusReceiver struct {
		InstanceID string     `yaml:"instance_id" mapstructure:"instance_id"`
		PlusAPI    APIDetails `yaml:"api_details" mapstructure:"api_details"`
	}

	ContainerMetricsReceiver struct {
		CollectionInterval time.Duration `yaml:"-" mapstructure:"collection_interval"`
	}

	HostMetrics struct {
		Scrapers           *HostMetricsScrapers `yaml:"scrapers"            mapstructure:"scrapers"`
		CollectionInterval time.Duration        `yaml:"collection_interval" mapstructure:"collection_interval"`
		InitialDelay       time.Duration        `yaml:"initial_delay"       mapstructure:"initial_delay"`
	}

	HostMetricsScrapers struct {
		CPU        *CPUScraper        `yaml:"cpu"        mapstructure:"cpu"`
		Disk       *DiskScraper       `yaml:"disk"       mapstructure:"disk"`
		Filesystem *FilesystemScraper `yaml:"filesystem" mapstructure:"filesystem"`
		Memory     *MemoryScraper     `yaml:"memory"     mapstructure:"memory"`
		Network    *NetworkScraper    `yaml:"network"    mapstructure:"network"`
	}
	CPUScraper        struct{}
	DiskScraper       struct{}
	FilesystemScraper struct{}
	MemoryScraper     struct{}
	NetworkScraper    struct{}

	Command struct {
		Server *ServerConfig `yaml:"server" mapstructure:"server"`
		Auth   *AuthConfig   `yaml:"auth"   mapstructure:"auth"`
		TLS    *TLSConfig    `yaml:"tls"    mapstructure:"tls"`
	}

	ServerConfig struct {
		Type ServerType `yaml:"type" mapstructure:"type"`
		Host string     `yaml:"host" mapstructure:"host"`
		Port int        `yaml:"port" mapstructure:"port"`
	}

	AuthConfig struct {
		Token     string `yaml:"token"     mapstructure:"token"`
		TokenPath string `yaml:"tokenpath" mapstructure:"tokenpath"`
	}

	TLSConfig struct {
		Cert       string `yaml:"cert"        mapstructure:"cert"`
		Key        string `yaml:"key"         mapstructure:"key"`
		Ca         string `yaml:"ca"          mapstructure:"ca"`
		ServerName string `yaml:"server_name" mapstructure:"server_name"`
		SkipVerify bool   `yaml:"skip_verify" mapstructure:"skip_verify"`
	}

	// Specialized TLS configuration for OtlpReceiver with self-signed cert generation.
	OtlpTLSConfig struct {
		Cert                   string `yaml:"cert"                      mapstructure:"cert"`
		Key                    string `yaml:"key"                       mapstructure:"key"`
		Ca                     string `yaml:"ca"                        mapstructure:"ca"`
		ServerName             string `yaml:"server_name"               mapstructure:"server_name"`
		ExistingCert           bool   `yaml:"-"`
		SkipVerify             bool   `yaml:"skip_verify"               mapstructure:"skip_verify"`
		GenerateSelfSignedCert bool   `yaml:"generate_self_signed_cert" mapstructure:"generate_self_signed_cert"`
	}

	Watchers struct {
		FileWatcher     FileWatcher     `yaml:"file_watcher"     mapstructure:"file_watcher"`
		InstanceWatcher InstanceWatcher `yaml:"instance_watcher" mapstructure:"instance_watcher"`
		// nolint: lll
		InstanceHealthWatcher InstanceHealthWatcher `yaml:"instance_health_watcher" mapstructure:"instance_health_watcher"`
	}

	InstanceWatcher struct {
		MonitoringFrequency time.Duration `yaml:"monitoring_frequency" mapstructure:"monitoring_frequency"`
	}

	InstanceHealthWatcher struct {
		MonitoringFrequency time.Duration `yaml:"monitoring_frequency" mapstructure:"monitoring_frequency"`
	}

	FileWatcher struct {
		ExcludeFiles        []string      `yaml:"exclude_files"        mapstructure:"exclude_files"`
		MonitoringFrequency time.Duration `yaml:"monitoring_frequency" mapstructure:"monitoring_frequency"`
	}
)

func (col *Collector) Validate(allowedDirectories []string) error {
	var err error
	cleaned := filepath.Clean(col.ConfigPath)

	if !isAllowedDir(cleaned, allowedDirectories) {
		err = errors.Join(err, fmt.Errorf("collector path %s not allowed", col.ConfigPath))
	}

	for _, nginxReceiver := range col.Receivers.NginxReceivers {
		err = errors.Join(err, nginxReceiver.Validate(allowedDirectories))
	}

	return err
}

func (nr *NginxReceiver) Validate(allowedDirectories []string) error {
	var err error
	if _, uuidErr := uuid.Parse(nr.InstanceID); uuidErr != nil {
		err = errors.Join(err, errors.New("invalid nginx receiver instance ID"))
	}

	for _, al := range nr.AccessLogs {
		if !isAllowedDir(al.FilePath, allowedDirectories) {
			err = errors.Join(err, fmt.Errorf("invalid nginx receiver access log path: %s", al.FilePath))
		}

		if len(al.FilePath) != 0 {
			// The log format's double quotes must be escaped so that
			// valid YAML is produced when executing the Go template.
			al.LogFormat = strings.ReplaceAll(al.LogFormat, `"`, `\"`)
		}
	}

	return err
}

func (c *Config) IsDirectoryAllowed(directory string) bool {
	return isAllowedDir(directory, c.AllowedDirectories)
}

func (c *Config) IsGrpcClientConfigured() bool {
	return c.Command != nil &&
		c.Command.Server != nil &&
		c.Command.Server.Host != "" &&
		c.Command.Server.Port != 0 &&
		c.Command.Server.Type == Grpc
}

func (c *Config) IsAuthConfigured() bool {
	return c.Command.Auth != nil &&
		(c.Command.Auth.Token != "" || c.Command.Auth.TokenPath != "")
}

func (c *Config) IsTLSConfigured() bool {
	return c.Command.TLS != nil
}

func (c *Config) IsFeatureEnabled(feature string) bool {
	for _, enabledFeature := range c.Features {
		if enabledFeature == feature {
			return true
		}
	}

	return false
}

func (c *Config) IsACollectorExporterConfigured() bool {
	if c.Collector == nil {
		return false
	}

	return c.Collector.Exporters.PrometheusExporter != nil ||
		c.Collector.Exporters.OtlpExporters != nil ||
		c.Collector.Exporters.Debug != nil
}

// nolint: cyclop, revive
func (c *Config) AreReceiversConfigured() bool {
	if c.Collector == nil {
		return false
	}

	return c.Collector.Receivers.NginxPlusReceivers != nil ||
		len(c.Collector.Receivers.NginxPlusReceivers) > 0 ||
		c.Collector.Receivers.OtlpReceivers != nil ||
		len(c.Collector.Receivers.OtlpReceivers) > 0 ||
		c.Collector.Receivers.NginxReceivers != nil ||
		len(c.Collector.Receivers.NginxReceivers) > 0 ||
		c.Collector.Receivers.HostMetrics != nil ||
		c.Collector.Receivers.ContainerMetrics != nil ||
		c.Collector.Receivers.TcplogReceivers != nil ||
		len(c.Collector.Receivers.TcplogReceivers) > 0
}

func isAllowedDir(dir string, allowedDirs []string) bool {
	for _, allowedDirectory := range allowedDirs {
		if strings.HasPrefix(dir, allowedDirectory) {
			return true
		}
	}

	return false
}
