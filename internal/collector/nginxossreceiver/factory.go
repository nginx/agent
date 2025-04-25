// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginxossreceiver

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/scraper"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/accesslog"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/stubstatus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

// nolint: ireturn
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		config.CreateDefaultConfig,
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
	)
}

// nolint: ireturn
func createMetrics(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*config.Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config")
	}

	var controllers []scraperhelper.ControllerOption

	stubStatusScraper := stubstatus.NewScraper(params, cfg)
	stubStatusMetrics, stubStatusMetricsError := scraper.NewMetrics(
		stubStatusScraper.Scrape,
		scraper.WithStart(stubStatusScraper.Start),
		scraper.WithShutdown(stubStatusScraper.Shutdown),
	)
	if stubStatusMetricsError != nil {
		return nil, stubStatusMetricsError
	}

	controllers = append(controllers, scraperhelper.AddScraper(metadata.Type, stubStatusMetrics))

	if len(cfg.AccessLogs) > 0 {
		accessLogScraper := accesslog.NewScraper(params, cfg)

		accessLogMetrics, accessLogMetricsError := scraper.NewMetrics(
			accessLogScraper.Scrape,
			scraper.WithStart(accessLogScraper.Start),
			scraper.WithShutdown(accessLogScraper.Shutdown),
		)
		if accessLogMetricsError != nil {
			return nil, accessLogMetricsError
		}

		controllers = append(controllers, scraperhelper.AddScraper(metadata.Type, accessLogMetrics))
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		cons,
		controllers...,
	)
}
