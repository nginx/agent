// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
)

const defaultCollectInterval = 10 * time.Second

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errors.New("endpoint cannot be empty for nginxplusreceiver")
	}

	if cfg.CollectionInterval == 0 {
		cfg.CollectionInterval = defaultCollectInterval
	}

	return nil
}
