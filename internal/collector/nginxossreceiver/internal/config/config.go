// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nginxreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
)

const (
	defaultCollectInterval = 10 * time.Second
	defaultClientTimeout   = 10 * time.Second
)

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	APIDetails                     APIDetails                    `mapstructure:"api_details"`
	AccessLogs                     []AccessLog                   `mapstructure:"access_logs"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

type APIDetails struct {
	URL      string `mapstructure:"url"`
	Listen   string `mapstructure:"listen"`
	Location string `mapstructure:"location"`
	Ca       string `mapstructure:"ca"`
}

type AccessLog struct {
	LogFormat string `mapstructure:"log_format"`
	FilePath  string `mapstructure:"file_path"`
}

// nolint: ireturn
func CreateDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = defaultCollectInterval

	return &Config{
		ControllerConfig: cfg,
		ClientConfig: confighttp.ClientConfig{
			Timeout: defaultClientTimeout,
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		AccessLogs:           []AccessLog{},
		APIDetails: APIDetails{
			URL:      "http://localhost:80/status",
			Listen:   "localhost:80",
			Location: "status",
			Ca:       "",
		},
	}
}
