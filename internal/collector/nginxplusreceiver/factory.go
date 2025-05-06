// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

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

	nps := newNginxPlusScraper(params, cfg)
	npsMetrics, npsMetricsError := scraper.NewMetrics(
		nps.Scrape,
		scraper.WithStart(nps.Start),
		scraper.WithShutdown(nps.Shutdown),
	)
	if npsMetricsError != nil {
		return nil, npsMetricsError
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		metricsConsumer,
		scraperhelper.AddScraper(metadata.Type, npsMetrics),
	)
}
