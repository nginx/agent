/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"time"

	"github.com/nginx/agent/sdk/v2/backoff"
)

type Config struct {
	Version               string
	Path                  string              `yaml:"-"`
	DynamicConfigPath     string              `mapstructure:"dynamic_config_path" yaml:"-"`
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
	IgnoreDirectives      []string            `mapstructure:"ignore_directives" yaml:"ignore_directives,omitempty"`
	QueueSize             int                 `mapstructure:"queue_size" yaml:"queue_size,omitempty"`
}

func (c *Config) IsGrpcServerConfigured() bool {
	return c.Server.Host != "" && c.Server.GrpcPort != 0
}

func (c *Config) IsFeatureEnabled(feature string) bool {
	for _, configFeature := range c.Features {
		if configFeature == feature {
			return true
		}
	}
	return false
}

func (c *Config) IsExtensionEnabled(extension string) bool {
	for _, configExtension := range c.Extensions {
		if configExtension == extension {
			return true
		}
	}
	return false
}

func (c *Config) GetServerBackoffSettings() backoff.BackoffSettings {
	return backoff.BackoffSettings{
		InitialInterval: c.Server.Backoff.InitialInterval,
		MaxInterval:     c.Server.Backoff.MaxInterval,
		MaxElapsedTime:  c.Server.Backoff.MaxElapsedTime,
		Multiplier:      c.Server.Backoff.Multiplier,
		Jitter:          c.Server.Backoff.RandomizationFactor,
	}
}

func (c *Config) GetMetricsBackoffSettings() backoff.BackoffSettings {
	return backoff.BackoffSettings{
		InitialInterval: c.AgentMetrics.Backoff.InitialInterval,
		MaxInterval:     c.AgentMetrics.Backoff.MaxInterval,
		MaxElapsedTime:  c.AgentMetrics.Backoff.MaxElapsedTime,
		Multiplier:      c.AgentMetrics.Backoff.Multiplier,
		Jitter:          c.AgentMetrics.Backoff.RandomizationFactor,
	}
}

type Server struct {
	Host     string `mapstructure:"host" yaml:"-"`
	GrpcPort int    `mapstructure:"grpcPort" yaml:"-"`
	Token    string `mapstructure:"token" yaml:"-"`
	Metrics  string `mapstructure:"metrics" yaml:"-"`
	Command  string `mapstructure:"command" yaml:"-"`
	// This is internal and shouldn't be exposed as a flag
	Target  string  `mapstructure:"target" yaml:"-"`
	Backoff Backoff `mapstructure:"backoff" yaml:"-"`
}

type Backoff struct {
	InitialInterval     time.Duration `mapstructure:"initial_interval" yaml:"-"`
	RandomizationFactor float64       `mapstructure:"randomization_factor" yaml:"-"`
	Multiplier          float64       `mapstructure:"multiplier" yaml:"-"`
	MaxInterval         time.Duration `mapstructure:"max_interval" yaml:"-"`
	MaxElapsedTime      time.Duration `mapstructure:"max_elapsed_time" yaml:"-"`
}

type AgentAPI struct {
	Host string `mapstructure:"host" yaml:"-"`
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
	ExcludeLogs                  string        `mapstructure:"exclude_logs" yaml:"-"`
	Debug                        bool          `mapstructure:"debug" yaml:"-"`
	NginxCountingSocket          string        `mapstructure:"socket" yaml:"-"`
	NginxClientVersion           int           `mapstructure:"client_version" yaml:"-"`
	ConfigReloadMonitoringPeriod time.Duration `mapstructure:"config_reload_monitoring_period" yaml:"-"`
	TreatWarningsAsErrors        bool          `mapstructure:"treat_warnings_as_errors" yaml:"-"`
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
	Backoff 		   Backoff       `mapstructure:"backoff" yaml:"-"`
}
