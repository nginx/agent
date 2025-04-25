// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"errors"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper"
	"go.opentelemetry.io/collector/scraper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
)

// nolint: ireturn
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

// nolint: ireturn
func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*config.Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config failed")
	}

	cpuScraper := cpuscraper.NewScraper(params, cpuscraper.NewConfig(cfg))
	cpuScraperMetrics, cpuScraperMetricsError := scraper.NewMetrics(
		cpuScraper.Scrape,
		scraper.WithStart(cpuScraper.Start),
		scraper.WithShutdown(cpuScraper.Shutdown),
	)
	if cpuScraperMetricsError != nil {
		return nil, cpuScraperMetricsError
	}

	memoryScraper := memoryscraper.NewScraper(params, memoryscraper.NewConfig(cfg))
	memoryScraperMetrics, memoryScraperMetricsError := scraper.NewMetrics(
		memoryScraper.Scrape,
		scraper.WithStart(memoryScraper.Start),
		scraper.WithShutdown(memoryScraper.Shutdown),
	)
	if memoryScraperMetricsError != nil {
		return nil, memoryScraperMetricsError
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		cons,
		scraperhelper.AddScraper(metadata.Type, cpuScraperMetrics),
		scraperhelper.AddScraper(metadata.Type, memoryScraperMetrics),
	)
}
