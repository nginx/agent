// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	supportedExporters = map[string]struct{}{
		"debug":      {},
		"otlp":       {},
		"prometheus": {},
	}

	supportedProcessors = map[string]struct{}{
		"batch": {},
	}

	supportedReceivers = map[string]struct{}{
		"otlp":        {},
		"hostmetrics": {},
		"nginx":       {},
		"prometheus":  {},
	}
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
		UUID               string           `yaml:"-"`
		Version            string           `yaml:"-"`
		Path               string           `yaml:"-"`
		Log                *Log             `yaml:"-" mapstructure:"log"`
		DataPlaneConfig    *DataPlaneConfig `yaml:"-" mapstructure:"data_plane_config"`
		Client             *Client          `yaml:"-" mapstructure:"client"`
		ConfigDir          string           `yaml:"-" mapstructure:"config-dirs"`
		AllowedDirectories []string         `yaml:"-"`
		Collector          *Collector       `yaml:"-" mapstructure:"collector"`
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

	Collector struct {
		ConfigPath string        `yaml:"-" mapstructure:"config_path"`
		Exporters  []Exporter    `yaml:"-" mapstructure:"exporters"`
		Health     *ServerConfig `yaml:"-" mapstructure:"health"`
		Processors []Processor   `yaml:"-" mapstructure:"processors"`
		Receivers  []Receiver    `yaml:"-" mapstructure:"receivers"`
	}

	// OTel Collector Exporter configuration.
	Exporter struct {
		Type   string        `yaml:"-" mapstructure:"type"`
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		Auth   *AuthConfig   `yaml:"-" mapstructure:"auth"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
	}

	// OTel Collector Processor configuration.
	Processor struct {
		Type string `yaml:"-" mapstructure:"type"`
	}

	// OTel Collector Receiver configuration.
	Receiver struct {
		Type   string        `yaml:"-" mapstructure:"type"`
		Server *ServerConfig `yaml:"-" mapstructure:"server"`
		Auth   *AuthConfig   `yaml:"-" mapstructure:"auth"`
		TLS    *TLSConfig    `yaml:"-" mapstructure:"tls"`
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
)

func (col *Collector) Validate(allowedDirectories []string) error {
	cleaned := filepath.Clean(col.ConfigPath)

	if !isAllowedDir(cleaned, allowedDirectories) {
		return fmt.Errorf("collector path %s not allowed", col.ConfigPath)
	}

	for _, exp := range col.Exporters {
		t := strings.ToLower(exp.Type)

		if _, ok := supportedExporters[t]; !ok {
			return fmt.Errorf("unsupported exporter type: %s", exp.Type)
		}

		// normalize field too
		exp.Type = t
	}

	for _, proc := range col.Processors {
		t := strings.ToLower(proc.Type)

		if _, ok := supportedProcessors[t]; !ok {
			return fmt.Errorf("unsupported processor type: %s", proc.Type)
		}

		proc.Type = t
	}

	for _, rec := range col.Receivers {
		t := strings.ToLower(rec.Type)

		if _, ok := supportedReceivers[t]; !ok {
			return fmt.Errorf("unsupported receiver type: %s", rec.Type)
		}

		rec.Type = t
	}

	return nil
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
