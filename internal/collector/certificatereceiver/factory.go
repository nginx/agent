// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"errors"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

//nolint:ireturn // must return metrics receiver interface
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

//nolint:ireturn // must return metrics receiver interface
func createMetricsReceiver(
	ctx context.Context,
	params receiver.Settings,
	rConf component.Config,
	metricsConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	logger := params.Logger.Sugar()

	logger.Info("Creating new certificate metrics receiver")

	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errors.New("failed to cast to Config in certificate metrics receiver")
	}

	cs := newCertificateScraper(params, cfg)
	csMetrics, csMetricsError := scraper.NewMetrics(
		cs.Scrape,
		scraper.WithStart(cs.Start),
		scraper.WithShutdown(cs.Shutdown),
	)
	if csMetricsError != nil {
		return nil, csMetricsError
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		metricsConsumer,
		scraperhelper.AddMetricsScraper(metadata.Type, csMetrics),
	)
}
