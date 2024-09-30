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

type ServerType int

const (
	Grpc ServerType = iota + 1
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
		Command            *Command         `yaml:"-" mapstructure:"command"`
		Log                *Log             `yaml:"-" mapstructure:"log"`
		DataPlaneConfig    *DataPlaneConfig `yaml:"-" mapstructure:"data_plane_config"`
		Client             *Client          `yaml:"-" mapstructure:"client"`
		Collector          *Collector       `yaml:"-" mapstructure:"collector"`
		File               *File            `yaml:"-" mapstructure:"file"`
		Common             *CommonSettings  `yaml:"-"`
		Watchers           *Watchers        `yaml:"-"`
		Version            string           `yaml:"-"`
		Path               string           `yaml:"-"`
		ConfigDir          string           `yaml:"-" mapstructure:"config-dirs"`
		UUID               string           `yaml:"-"`
		AllowedDirectories []string         `yaml:"-"`
		Features           []string         `yaml:"-"`
	}

	Log struct {
		Level string `yaml:"-" mapstructure:"level"`
		Path  string `yaml:"-" mapstructure:"path"`
	}

	DataPlaneConfig struct {
		Nginx *NginxDataPlaneConfig `yaml:"-" mapstructure:"nginx"`
	}

	NginxDataPlaneConfig struct {
		ExcludeLogs            string        `yaml:"-" mapstructure:"exclude_logs"`
		ReloadMonitoringPeriod time.Duration `yaml:"-" mapstructure:"reload_monitoring_period"`
		TreatWarningsAsErrors  bool          `yaml:"-" mapstructure:"treat_warnings_as_errors"`
	}

	Client struct {
		Timeout             time.Duration `yaml:"-" mapstructure:"timeout"`
		Time                time.Duration `yaml:"-" mapstructure:"time"`
		PermitWithoutStream bool          `yaml:"-" mapstructure:"permit_without_stream"`
		// if MaxMessageSize is size set then we use that value,
		// otherwise MaxMessageRecieveSize and MaxMessageSendSize for individual settings
		MaxMessageSize        int `yaml:"-" mapstructure:"max_message_size"`
		MaxMessageRecieveSize int `yaml:"-" mapstructure:"max_message_receive_size"`
		MaxMessageSendSize    int `yaml:"-" mapstructure:"max_message_send_size"`
	}

	Collector struct {
		ConfigPath string     `yaml:"-" mapstructure:"config_path"`
		Log        *Log       `yaml:"-" mapstructure:"log"`
		Exporters  Exporters  `yaml:"-" mapstructure:"exporters"`
		Extensions Extensions `yaml:"-" mapstructure:"extensions"`
		Processors Processors `yaml:"-" mapstructure:"processors"`
		Receivers  Receivers  `yaml:"-" mapstructure:"receivers"`
	}

	Exporters struct {
		Debug              *DebugExporter      `yaml:"-" mapstructure:"debug"`
		PrometheusExporter *PrometheusExporter `yaml:"-" mapstructure:"prometheus_exporter"`
		OtlpExporters      []OtlpExporter      `yaml:"-" mapstructure:"otlp_exporters"`
	}

	OtlpExporter struct {
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		Auth   *AuthConfig   `yaml:"-" mapstructure:"auth"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
	}

	Extensions struct {
		Health *Health `yaml:"-" mapstructure:"health"`
	}

	Health struct {
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
		Path   string        `yaml:"-" mapstructure:"path"`
	}

	DebugExporter struct{}

	PrometheusExporter struct {
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
	}

	// OTel Collector Processors configuration.
	Processors struct {
		Batch *Batch `yaml:"-" mapstructure:"batch"`
	}

	Batch struct {
		SendBatchSize    uint32        `yaml:"-" mapstructure:"send_batch_size"`
		SendBatchMaxSize uint32        `yaml:"-" mapstructure:"send_batch_max_size"`
		Timeout          time.Duration `yaml:"-" mapstructure:"timeout"`
	}

	// OTel Collector Receiver configuration.
	Receivers struct {
		OtlpReceivers      []OtlpReceiver      `yaml:"-" mapstructure:"otlp_receivers"`
		NginxReceivers     []NginxReceiver     `yaml:"-" mapstructure:"nginx_receivers"`
		NginxPlusReceivers []NginxPlusReceiver `yaml:"-" mapstructure:"nginx_plus_receivers"`
		HostMetrics        HostMetrics         `yaml:"-" mapstructure:"host_metrics"`
	}

	OtlpReceiver struct {
		Server        *ServerConfig  `yaml:"-" mapstructure:"server"`
		Auth          *AuthConfig    `yaml:"-" mapstructure:"auth"`
		OtlpTLSConfig *OtlpTLSConfig `yaml:"-" mapstructure:"tls"`
	}

	NginxReceiver struct {
		InstanceID string      `yaml:"-" mapstructure:"instance_id"`
		StubStatus string      `yaml:"-" mapstructure:"stub_status"`
		AccessLogs []AccessLog `yaml:"-" mapstructure:"access_logs"`
	}

	AccessLog struct {
		FilePath  string `yaml:"-" mapstructure:"file_path"`
		LogFormat string `yaml:"-" mapstructure:"log_format"`
	}

	NginxPlusReceiver struct {
		InstanceID string `yaml:"-" mapstructure:"instance_id"`
		PlusAPI    string `yaml:"-" mapstructure:"plus_api"`
	}

	HostMetrics struct {
		Scrapers           *HostMetricsScrapers `yaml:"-" mapstructure:"scrapers"`
		CollectionInterval time.Duration        `yaml:"-" mapstructure:"collection_interval"`
		InitialDelay       time.Duration        `yaml:"-" mapstructure:"initial_delay"`
	}

	HostMetricsScrapers struct {
		CPU        *CPUScraper        `yaml:"-" mapstructure:"cpu"`
		Disk       *DiskScraper       `yaml:"-" mapstructure:"disk"`
		Filesystem *FilesystemScraper `yaml:"-" mapstructure:"filesystem"`
		Memory     *MemoryScraper     `yaml:"-" mapstructure:"memory"`
		Network    *NetworkScraper    `yaml:"-" mapstructure:"network"`
	}
	CPUScraper        struct{}
	DiskScraper       struct{}
	FilesystemScraper struct{}
	MemoryScraper     struct{}
	NetworkScraper    struct{}

	GRPC struct {
		Target         string        `yaml:"-" mapstructure:"target"`
		ConnTimeout    time.Duration `yaml:"-" mapstructure:"connection_timeout"`
		MinConnTimeout time.Duration `yaml:"-" mapstructure:"minimum_connection_timeout"`
		BackoffDelay   time.Duration `yaml:"-" mapstructure:"backoff_delay"`
	}

	Command struct {
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		Auth   *AuthConfig   `yaml:"-" mapstructure:"auth"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
	}

	ServerConfig struct {
		Host string     `yaml:"-" mapstructure:"host"`
		Port int        `yaml:"-" mapstructure:"port"`
		Type ServerType `yaml:"-" mapstructure:"type"`
	}

	AuthConfig struct {
		Token string `yaml:"-" mapstructure:"token"`
	}

	TLSConfig struct {
		Cert       string `yaml:"-" mapstructure:"cert"`
		Key        string `yaml:"-" mapstructure:"key"`
		Ca         string `yaml:"-" mapstructure:"ca"`
		ServerName string `yaml:"-" mapstructure:"server_name"`
		SkipVerify bool   `yaml:"-" mapstructure:"skip_verify"`
	}

	// Specialized TLS configuration for OtlpReceiver with self-signed cert generation.
	OtlpTLSConfig struct {
		Cert                   string `yaml:"-" mapstructure:"cert"`
		Key                    string `yaml:"-" mapstructure:"key"`
		Ca                     string `yaml:"-" mapstructure:"ca"`
		ServerName             string `yaml:"-" mapstructure:"server_name"`
		ExistingCert           bool   `yaml:"-"`
		SkipVerify             bool   `yaml:"-" mapstructure:"skip_verify"`
		GenerateSelfSignedCert bool   `yaml:"-" mapstructure:"generate_self_signed_cert"`
	}

	File struct {
		Location string `yaml:"-" mapstructure:"location"`
	}

	CommonSettings struct {
		InitialInterval     time.Duration `yaml:"-" mapstructure:"initial_interval"`
		MaxInterval         time.Duration `yaml:"-" mapstructure:"max_interval"`
		MaxElapsedTime      time.Duration `yaml:"-" mapstructure:"max_elapsed_time"`
		RandomizationFactor float64       `yaml:"-" mapstructure:"randomization_factor"`
		Multiplier          float64       `yaml:"-" mapstructure:"multiplier"`
	}

	Watchers struct {
		InstanceWatcher       InstanceWatcher       `yaml:"-" mapstructure:"instance_watcher"`
		InstanceHealthWatcher InstanceHealthWatcher `yaml:"-" mapstructure:"instance_health_watcher"`
		FileWatcher           FileWatcher           `yaml:"-" mapstructure:"file_watcher"`
	}

	InstanceWatcher struct {
		MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
	}

	InstanceHealthWatcher struct {
		MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
	}

	FileWatcher struct {
		MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
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

func isAllowedDir(dir string, allowedDirs []string) bool {
	for _, allowedDirectory := range allowedDirs {
		if strings.HasPrefix(dir, allowedDirectory) {
			return true
		}
	}

	return false
}
