// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package scraper

import (
	"context"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type ContainerScraper struct {
	logger *zap.Logger
	//mb       *metadata.MetricsBuilder
	settings receiver.Settings
}

func NewContainerScraper(
	settings receiver.Settings,
	logger *zap.Logger,
) (*ContainerScraper, error) {
	logger = settings.Logger
	logger.Info("Creating container metrics scraper")
	return &ContainerScraper{
		logger:   logger,
		settings: settings,
	}, nil
}

func (cms *ContainerScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (cms *ContainerScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (cms *ContainerScraper) Start(parentCtx context.Context, _ component.Host) error {
	cms.logger.Info("Container metrics scraper started")
	//ctx, cancel := context.WithCancel(parentCtx)
	//cms.cancel = cancel
	//
	//err := cms.pipe.Start(storage.NewNopClient())
	//if err != nil {
	//	return fmt.Errorf("stanza pipeline start: %w", err)
	//}
	//
	//cms.wg.Add(1)
	//go cms.runConsumer(ctx)

	return nil
}

func (cms *ContainerScraper) Scrape(
	ctx context.Context,
) (pmetric.Metrics, error) {
	return pmetric.Metrics{}, nil
}

func (cms *ContainerScraper) recordMetrics() {}
