// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"time"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/cgroup"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
) *containerScraper {
	logger := settings.Logger
	logger.Info("Creating container metrics scraper")

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &containerScraper{
		logger:   logger,
		settings: settings,
		mb:       mb,
		rb:       rb,
	}
}

func (cms *containerScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (cms *containerScraper) scrape(
	ctx context.Context,
) (pmetric.Metrics, error) {
	cms.logger.Debug("Starting container metrics scrape")

	// set resource attributes
	// cms.rb.SetResourceID()

	// record metrics
	cms.recordMetrics()

	cms.logger.Debug("Finished container metrics scrape, emitting metrics")

	return cms.mb.Emit(metadata.WithResource(cms.rb.Emit())), nil
}

func (cms *containerScraper) recordMetrics() {
	cms.logger.Debug("Collecting container metrics")
	_ = pcommon.NewTimestampFromTime(time.Now())

	// collect cpu
	_ = cgroup.NewCgroupCPUSource()

	// collect memory
	_ = cgroup.NewCgroupMemorySource()
	// cms.mb.RecordContainerMemoryCurrentDataPoint()
}
