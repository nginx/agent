// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import "time"

type Config struct {
	Version            string         `yaml:"-"`
	Path               string         `yaml:"-"`
	Log                Log            `yaml:"-" mapstructure:"log"`
	ProcessMonitor     ProcessMonitor `yaml:"-" mapstructure:"process_monitor"`
	DataPlaneAPI       DataPlaneAPI   `yaml:"-" mapstructure:"data_plane_api"`
	Client             Client         `yaml:"-" mapstructure:"client"`
	ConfigDir          string         `yaml:"-" mapstructure:"config-dirs"`
	AllowedDirectories []string       `yaml:"-"`
}

type Log struct {
	Level string `yaml:"-" mapstructure:"level"`
	Path  string `yaml:"-" mapstructure:"path"`
}

type ProcessMonitor struct {
	MonitoringFrequency time.Duration `yaml:"-" mapstructure:"monitoring_frequency"`
}

type DataPlaneAPI struct {
	Host string `yaml:"-" mapstructure:"host"`
	Port int    `yaml:"-" mapstructure:"port"`
}

type Client struct {
	Timeout time.Duration `yaml:"-" mapstructure:"timeout"`
}
