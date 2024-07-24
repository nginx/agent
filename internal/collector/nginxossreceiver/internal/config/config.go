// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	confighttp.ClientConfig        `mapstructure:",squash"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
	// Configures `stanza`
	BaseConfig      adapter.BaseConfig `mapstructure:",squash"`
	InputConfig     file.Config        `mapstructure:",squash"`
	NginxConfigPath string             `mapstructure:"nginx_conf_path"`
}

func CreateDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = 10 * time.Second

	return &Config{
		ControllerConfig: cfg,
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "http://localhost:80/status",
			Timeout:  10 * time.Second,
		},
		InputConfig:          *file.NewConfig(),
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		NginxConfigPath:      "",
	}
}
