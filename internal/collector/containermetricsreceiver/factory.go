// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package containermetricsreceiver

import (
	"context"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type Config struct {
	confighttp.ClientConfig        `mapstructure:",squash"`
	scraperhelper.ControllerConfig `mapstructure:",squash"`
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.Type{},
		config.CreateDefaultConfig,
		//receiver.WithMetrics(
		//	createMetricsReceiver,
		//	metadata.MetricsStability,
		//),
	)
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	metricsConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	return nil, nil
}
