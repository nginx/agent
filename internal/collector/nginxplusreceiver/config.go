// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
)

const defaultCollectInterval = 10 * time.Second

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	APIDetails                     APIDetails                    `mapstructure:"api_details"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

type APIDetails struct {
	URL      string `mapstructure:"url"`
	Listen   string `mapstructure:"listen"`
	Location string `mapstructure:"location"`
	Ca       string `mapstructure:"ca"`
}

// Validate checks if the receiver configuration is valid
// nolint: ireturn
func (cfg *Config) Validate() error {
	if cfg.APIDetails.URL == "" {
		return errors.New("endpoint cannot be empty for nginxplusreceiver")
	}

	if cfg.CollectionInterval == 0 {
		cfg.CollectionInterval = defaultCollectInterval
	}

	return nil
}

// nolint: ireturn
func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = defaultCollectInterval

	return &Config{
		ControllerConfig: cfg,
		ClientConfig: confighttp.ClientConfig{
			Timeout: defaultTimeout,
		},
		APIDetails: APIDetails{
			URL:      "http://localhost:80/api",
			Listen:   "localhost:80",
			Location: "/api",
			Ca:       "",
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}
