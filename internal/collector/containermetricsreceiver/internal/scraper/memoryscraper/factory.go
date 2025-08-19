// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package memoryscraper

import (
	"context"
	"errors"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/scraper"
)

// NewFactory for CPU scraper.
//
//nolint:ireturn // must return a CPU scraper
func NewFactory() scraper.Factory {
	return scraper.NewFactory(
		metadata.Type,
		createDefaultConfig,
		scraper.WithMetrics(createMetricsScraper, metadata.MetricsStability),
	)
}

// createDefaultConfig creates the default configuration for the Scraper.
//
//nolint:ireturn // must return a default configuration for scraper
func createDefaultConfig() component.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}

// createMetricsScraper creates a scraper based on provided config.
//
//nolint:ireturn // must return a metric scraper interface
func createMetricsScraper(
	ctx context.Context,
	settings scraper.Settings,
	config component.Config,
) (scraper.Metrics, error) {
	cfg, ok := config.(*Config)
	if !ok {
		return nil, errors.New("cast to metrics scraper config")
	}

	s := NewScraper(ctx, settings, cfg)

	return scraper.NewMetrics(
		s.Scrape,
		scraper.WithStart(s.Start),
	)
}
