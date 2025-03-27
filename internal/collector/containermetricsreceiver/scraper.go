// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type containerScraper struct {
	logger   *zap.Logger
	mb       *metadata.MetricsBuilder
	rb       *metadata.ResourceBuilder
	settings receiver.Settings
}

func newContainerScraper(
	settings receiver.Settings,
	cfg *config.Config,
) (*containerScraper, error) {
	logger := settings.Logger
	logger.Info("Creating container metrics scraper")

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &containerScraper{
		logger:   logger,
		settings: settings,
		mb:       mb,
		rb:       rb,
	}, nil
}

func (cms *containerScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (cms *containerScraper) scrape(
	ctx context.Context,
) (pmetric.Metrics, error) {

	// set resource attributes
	//cms.rb.SetResourceID()

	// collect metrics
	// record metrics

	return cms.mb.Emit(metadata.WithResource(cms.rb.Emit())), nil
}

func (cms *containerScraper) recordMetrics() {

}
