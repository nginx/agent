// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package memoryscraper

import (
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/metadata"
)

type Config struct {
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

func NewConfig(cfg *config.Config) *Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		ControllerConfig: scraperhelper.ControllerConfig{
			CollectionInterval: cfg.CollectionInterval,
			InitialDelay:       cfg.InitialDelay,
			Timeout:            cfg.Timeout,
		},
	}
}
