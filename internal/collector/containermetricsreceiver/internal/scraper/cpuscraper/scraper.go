// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cpuscraper

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/scraper"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper/internal/cgroup"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper/internal/metadata"
)

var basePath = "/sys/fs/cgroup/"

type CPUScraper struct {
	cfg       *Config
	mb        *metadata.MetricsBuilder
	rb        *metadata.ResourceBuilder
	cpuSource *cgroup.CPUSource
	settings  scraper.Settings
}

func NewScraper(
	_ context.Context,
	settings scraper.Settings,
	cfg *Config,
) *CPUScraper {
	logger := settings.Logger
	logger.Info("Creating container CPU scraper")

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
	s.settings.Logger.Info("Starting container CPU scraper")
	s.cpuSource = cgroup.NewCPUSource(basePath)

	return nil
}

func (s *CPUScraper) Scrape(context.Context) (pmetric.Metrics, error) {
	s.settings.Logger.Debug("Scraping container CPU metrics")

	now := pcommon.NewTimestampFromTime(time.Now())

	stats, err := s.cpuSource.Collect()
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	s.settings.Logger.Debug("Collected container CPU metrics", zap.Any("cpu", stats))

	s.mb.RecordSystemCPULogicalCountDataPoint(now, int64(stats.NumberOfLogicalCPUs))
	s.mb.RecordSystemCPUUtilizationDataPoint(now, stats.User, metadata.AttributeStateUser)
	s.mb.RecordSystemCPUUtilizationDataPoint(now, stats.System, metadata.AttributeStateSystem)

	return s.mb.Emit(metadata.WithResource(s.rb.Emit())), nil
}
