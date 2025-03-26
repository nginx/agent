package containermetricsreceiver

// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

import (
	"context"
	"errors"
	"fmt"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"time"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
)

const defaultTimeout = 10 * time.Second

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		config.CreateDefaultConfig,
		receiver.WithMetrics(
			createMetricsReceiver,
			metadata.MetricsStability,
		),
	)
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {

	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config failed")
	}

	containerScraper, err := scraper.NewContainerScraper(params, params.Logger)
	if err != nil {
		return nil, fmt.Errorf("new container scraper: %w", err)
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ControllerConfig,
		params,
		cons,
		scraperhelper.AddScraperWithType(metadata.Type, containerScraper),
	)
}
