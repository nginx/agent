// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"strings"
	"time"
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

const (
	Unsupported OTelReceiver = "unknown"
	OTLP        OTelReceiver = "otlp"
	HostMetrics OTelReceiver = "hostmetrics"
)

type (
	Config struct {
		UUID               string           `yaml:"-"`
		Version            string           `yaml:"-"`
		Path               string           `yaml:"-"`
		Log                *Log             `yaml:"-" mapstructure:"log"`
		DataPlaneConfig    *DataPlaneConfig `yaml:"-" mapstructure:"data_plane_config"`
		Client             *Client          `yaml:"-" mapstructure:"client"`
		ConfigDir          string           `yaml:"-" mapstructure:"config-dirs"`
		AllowedDirectories []string         `yaml:"-"`
		Metrics            *Metrics         `yaml:"-" mapstructure:"metrics"`
		Command            *Command         `yaml:"-" mapstructure:"command"`
		File               *File            `yaml:"-" mapstructure:"file"`
		Common             *CommonSettings  `yaml:"-"`
		Watchers           *Watchers        `yaml:"-"`
	}

	Log struct {
		Level string `yaml:"-" mapstructure:"level"`
		Path  string `yaml:"-" mapstructure:"path"`
	}

	DataPlaneConfig struct {
		Nginx *NginxDataPlaneConfig `yaml:"-" mapstructure:"nginx"`
	}

	NginxDataPlaneConfig struct {
		ReloadMonitoringPeriod time.Duration `yaml:"-" mapstructure:"reload_monitoring_period"`
		TreatWarningsAsError   bool          `yaml:"-" mapstructure:"treat_warnings_as_error"`
	}

	Client struct {
		Timeout             time.Duration `yaml:"-" mapstructure:"timeout"`
		Time                time.Duration `yaml:"-" mapstructure:"time"`
		PermitWithoutStream bool          `yaml:"-" mapstructure:"permit_without_stream"`
	}

	Metrics struct {
		Collector           bool           `yaml:"-" mapstructure:"collector"`
		OTLPExportURL       string         `yaml:"-" mapstructure:"otlp_export_url"`
		OTLPReceiverURL     string         `yaml:"-" mapstructure:"otlp_receiver_port"`
		CollectorConfigPath string         `yaml:"-" mapstructure:"collector_config_path"`
		CollectorReceivers  []OTelReceiver `yaml:"-" mapstructure:"collector_receivers"`
	}

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
		SkipVerify bool   `yaml:"-" mapstructure:"skip_verify"`
		ServerName string `yaml:"-" mapstructure:"server_name"`
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
	}

	InstanceWatcher struct {
		MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
	}

	InstanceHealthWatcher struct {
		MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
	}

	// Enum for the supported OTel Collector receiver names that the Agent supports.
	OTelReceiver string
)

func (c *Config) IsDirectoryAllowed(directory string) bool {
	for _, allowedDirectory := range c.AllowedDirectories {
		if strings.HasPrefix(directory, allowedDirectory) {
			return true
		}
	}

	return false
}

// Converts a string to a OTelReceiver
func toOTelReceiver(input string) OTelReceiver {
	switch OTelReceiver(input) {
	case OTLP, HostMetrics:
		return toOTelReceiver(input)
	case Unsupported:
		fallthrough
	default:
		return Unsupported
	}
}
