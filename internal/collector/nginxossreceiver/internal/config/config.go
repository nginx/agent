// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nginxreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
)

const (
	defaultCollectInterval = 10 * time.Second
	defaultClientTimeout   = 10 * time.Second
)

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	AccessLogs                     []AccessLog `mapstructure:"access_logs"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
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
			Endpoint: "http://localhost:80/status",
			Timeout:  defaultClientTimeout,
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		AccessLogs:           []AccessLog{},
	}
}
