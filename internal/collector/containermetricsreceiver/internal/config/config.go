// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
)

type Config struct {
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

// nolint: ireturn
func CreateDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	return &Config{
		ControllerConfig:     cfg,
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}
