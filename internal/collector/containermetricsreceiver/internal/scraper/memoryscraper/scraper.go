// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package memoryscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/scraper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/cgroup"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var basePath = "/sys/fs/cgroup/"

type CPUScraper struct {
	cfg          *Config
	mb           *metadata.MetricsBuilder
	rb           *metadata.ResourceBuilder
	memorySource *cgroup.MemorySource
	settings     scraper.Settings
}

func NewScraper(
	_ context.Context,
	settings scraper.Settings,
	cfg *Config,
) *CPUScraper {
	logger := settings.Logger
	logger.Info("Creating container memory scraper")

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &CPUScraper{
		settings: settings,
		cfg:      cfg,
		mb:       mb,
		rb:       rb,
	}
}

func (s *CPUScraper) Start(_ context.Context, _ component.Host) error {
	s.settings.Logger.Info("Starting container memory scraper")
	s.memorySource = cgroup.NewMemorySource(basePath)

	return nil
}

func (s *CPUScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.settings.Logger.Debug("Scraping container memory metrics")

	now := pcommon.NewTimestampFromTime(time.Now())

	stats, err := s.memorySource.VirtualMemoryStatWithContext(ctx)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	s.settings.Logger.Debug("Collected container memory metrics", zap.Any("metrics", stats))

	s.mb.RecordSystemMemoryUsageDataPoint(now, int64(stats.Used), metadata.AttributeStateUsed)
	s.mb.RecordSystemMemoryUsageDataPoint(now, int64(stats.Free), metadata.AttributeStateFree)

	return s.mb.Emit(metadata.WithResource(s.rb.Emit())), nil
}
