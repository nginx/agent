/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import "time"

type Config struct {
	Version        string         `yaml:"-"`
	Path           string         `yaml:"-"`
	Log            Log            `mapstructure:"log" yaml:"-"`
	ProcessMonitor ProcessMonitor `mapstructure:"process_monitor" yaml:"-"`
	DataplaneAPI   DataplaneAPI   `mapstructure:"dataplane_api" yaml:"-"`
	Client         Client         `mapstructure:"client" yaml:"-"`
}

type Log struct {
	Level string `mapstructure:"level" yaml:"-"`
	Path  string `mapstructure:"path" yaml:"-"`
}

type ProcessMonitor struct {
	MonitoringFrequency time.Duration `mapstructure:"monitoring_frequency" yaml:"-"`
}

type DataplaneAPI struct {
	Host string `mapstructure:"host" yaml:"-"`
	Port int    `mapstructure:"port" yaml:"-"`
}

type Client struct {
	Timeout time.Duration `mapstructure:"timeout" yaml:"-"`
}
