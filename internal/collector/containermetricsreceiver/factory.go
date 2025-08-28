// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"errors"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
)

//nolint:ireturn // must return metrics interface
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

//nolint:ireturn // must return metrics interface
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

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		cons,
		scraperhelper.AddFactoryWithConfig(cpuscraper.NewFactory(), cpuscraper.NewConfig(cfg)),
		scraperhelper.AddFactoryWithConfig(memoryscraper.NewFactory(), memoryscraper.NewConfig(cfg)),
	)
}
