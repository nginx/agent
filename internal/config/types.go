// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"strings"
	"time"
)

type Config struct {
	UUID               string           `yaml:"-"`
	Version            string           `yaml:"-"`
	Path               string           `yaml:"-"`
	Log                *Log             `yaml:"-" mapstructure:"log"`
	ProcessMonitor     *ProcessMonitor  `yaml:"-" mapstructure:"process_monitor"`
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

type Log struct {
	Level string `yaml:"-" mapstructure:"level"`
	Path  string `yaml:"-" mapstructure:"path"`
}

type ProcessMonitor struct {
	MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
}

type DataPlaneConfig struct {
	Nginx *NginxDataPlaneConfig `yaml:"-" mapstructure:"nginx"`
}

type NginxDataPlaneConfig struct {
	ReloadMonitoringPeriod time.Duration `yaml:"-" mapstructure:"reload_monitoring_period"`
	TreatWarningsAsError   bool          `yaml:"-" mapstructure:"treat_warnings_as_error"`
}

type Client struct {
	Timeout             time.Duration `yaml:"-" mapstructure:"timeout"`
	Time                time.Duration `yaml:"-" mapstructure:"time"`
	PermitWithoutStream bool          `yaml:"-" mapstructure:"permit_without_stream"`
}

type Metrics struct {
	ProduceInterval  time.Duration     `yaml:"-" mapstructure:"produce_interval"`
	OTelExporter     *OTelExporter     `yaml:"-" mapstructure:"otel_exporter"`
	PrometheusSource *PrometheusSource `yaml:"-" mapstructure:"prometheus_source"`
	// temporary setting to enable the collector
	Collector bool `yaml:"-" mapstructure:"collector"`
}

// PrometheusSource is a DataSources implementation
type PrometheusSource struct {
	Endpoints []string `yaml:"-" mapstructure:"endpoints"`
}

// OTelExporter is an Exporters implementation
type OTelExporter struct {
	BufferLength     int           `yaml:"-" mapstructure:"buffer_length"`
	ExportRetryCount int           `yaml:"-" mapstructure:"export_retry_count"`
	ExportInterval   time.Duration `yaml:"-" mapstructure:"export_interval"`
	GRPC             *GRPC         `yaml:"-" mapstructure:"grpc"`
}

type GRPC struct {
	Target         string        `yaml:"-" mapstructure:"target"`
	ConnTimeout    time.Duration `yaml:"-" mapstructure:"connection_timeout"`
	MinConnTimeout time.Duration `yaml:"-" mapstructure:"minimum_connection_timeout"`
	BackoffDelay   time.Duration `yaml:"-" mapstructure:"backoff_delay"`
}

// Command Connection settings for connecting to a Command and Control Server
type Command struct {
	Server *ServerConfig `yaml:"-" mapstructure:"server"`
	Auth   *AuthConfig   `yaml:"-" mapstructure:"auth"`
	TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
}

type ServerConfig struct {
	Host string `yaml:"-" mapstructure:"host"`
	Port int    `yaml:"-" mapstructure:"port"`
	Type string `yaml:"-" mapstructure:"type"`
}

type AuthConfig struct {
	Token string `yaml:"-" mapstructure:"token"`
}

type TLSConfig struct {
	Cert       string `yaml:"-" mapstructure:"cert"`
	Key        string `yaml:"-" mapstructure:"key"`
	Ca         string `yaml:"-" mapstructure:"ca"`
	SkipVerify bool   `yaml:"-" mapstructure:"skip_verify"`
	ServerName string `yaml:"-" mapstructure:"server_name"`
}

type File struct {
	Location string `yaml:"-" mapstructure:"location"`
}

type CommonSettings struct {
	InitialInterval     time.Duration `yaml:"-" mapstructure:"initial_interval"`
	MaxInterval         time.Duration `yaml:"-" mapstructure:"max_interval"`
	MaxElapsedTime      time.Duration `yaml:"-" mapstructure:"max_elapsed_time"`
	RandomizationFactor float64       `yaml:"-" mapstructure:"randomization_factor"`
	Multiplier          float64       `yaml:"-" mapstructure:"multiplier"`
}

type Watchers struct {
	InstanceWatcher       InstanceWatcher       `yaml:"-" mapstructure:"instance_watcher"`
	InstanceHealthWatcher InstanceHealthWatcher `yaml:"-" mapstructure:"instance_health_watcher"`
}

type InstanceWatcher struct {
	MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
}

type InstanceHealthWatcher struct {
	MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
}

func (c *Config) IsDirectoryAllowed(directory string) bool {
	isDirectoryAllowed := false

	for _, allowedDirectory := range c.AllowedDirectories {
		if strings.HasPrefix(directory, allowedDirectory) {
			isDirectoryAllowed = true
			break
		}
	}

	return isDirectoryAllowed
}
