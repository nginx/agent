// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginxossreceiver

import (
	"context"
	"errors"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/accesslog"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/stubstatus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// nolint: ireturn
func NewFactory() receiver.Factory {
	stubStatusOption := receiver.WithMetrics(createStubStatusReceiver, metadata.MetricsStability)
	accessLogOptions := receiver.WithMetrics(createAccessLogReceiver, metadata.MetricsStability)

	return receiver.NewFactory(
		metadata.Type,
		config.CreateDefaultConfig,
		stubStatusOption,
		accessLogOptions,
	)
}

// nolint: ireturn
func createStubStatusReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*config.Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config")
	}

	return stubstatus.NewScraper(params, cfg), nil
}

// nolint: ireturn
func createAccessLogReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*config.Config)
	if !ok {
		return nil, errors.New("cast to metrics receiver config")
	}

	return accesslog.NewScraper(params, cfg)
}
