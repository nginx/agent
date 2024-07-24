// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nginxossreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nginxossreceiver"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/accesslog"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/stubstatus"
)

// NewFactory creates a factory for nginx receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		config.CreateDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*config.Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config")
	}

	logger := params.Logger.Sugar()

	ns := stubstatus.NewScraper(params, cfg)
	scraperOpts := []scraperhelper.ScraperControllerOption{
		scraperhelper.AddScraper(ns),
	}

	if cfg.NginxConfigPath != "" {
		nals, err := accesslog.NewScraper(params, cfg)
		if err != nil {
			logger.Errorf("Failed to initialize NGINX Access Log scraper: %s", err.Error())
		} else {
			scraperOpts = append(scraperOpts, scraperhelper.AddScraper(nals))
		}
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ControllerConfig, params, consumer,
		scraperOpts...,
	)
}
