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
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
)

const (
	defaultCollectInterval = 10 * time.Second
	defaultClientTimeout   = 10 * time.Second
)

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	InputConfig                    file.Config        `mapstructure:",squash"`
	AccessLogFormat                string             `mapstructure:"access_log_format"`
	BaseConfig                     adapter.BaseConfig `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
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
		InputConfig:          *file.NewConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		AccessLogFormat:      "",
	}
}
