// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
)

const defaultTimeout = 10 * time.Second

// nolint: ireturn
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

// nolint: ireturn
func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = defaultCollectInterval

	return &Config{
		ControllerConfig: cfg,
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "http://localhost:80/api",
			Timeout:  defaultTimeout,
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}

// nolint: ireturn
func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	rConf component.Config,
	metricsConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	logger := params.Logger.Sugar()

	logger.Info("Creating new NGINX Plus metrics receiver")

	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errors.New("failed to cast to Config in NGINX Plus metrics receiver")
	}

	nps, err := newNginxPlusScraper(params, cfg)
	if err != nil {
		return nil, fmt.Errorf("new nginx plus scraper: %w", err)
	}

	scraper, err := scraperhelper.NewScraper(
		metadata.Type.String(),
		nps.scrape,
		scraperhelper.WithShutdown(nps.Shutdown),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ControllerConfig, params, metricsConsumer,
		scraperhelper.AddScraper(scraper),
	)
}
