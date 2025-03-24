// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package scraper

import (
	"context"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type containerScraper struct {
	logger *zap.Logger
	//mb       *metadata.MetricsBuilder
	settings receiver.Settings
}

func newContainerScraper(
	settings receiver.Settings,
	logger *zap.Logger,
) (*containerScraper, error) {
	logger = settings.Logger
	logger.Info("Creating container metrics scraper")
	return &containerScraper{
		logger:   logger,
		settings: settings,
	}, nil
}

func (cms *containerScraper) scrape(
	ctx context.Context,
	settings receiver.Settings,
) (receiver.Metrics, error) {
	return nil, nil
}

func (cms *containerScraper) Shutdown() error {
	return nil
}

func (cms *containerScraper) recordMetrics() {}
