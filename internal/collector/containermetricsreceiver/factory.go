// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

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

	cs := newContainerScraper(params, cfg)

	scraper, err := scraperhelper.NewScraperWithoutType(
		cs.scrape,
		scraperhelper.WithShutdown(cs.Shutdown),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ControllerConfig,
		params,
		cons,
		scraperhelper.AddScraperWithType(metadata.Type, scraper),
	)
}
