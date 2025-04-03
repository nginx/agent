// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"context"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/cgroup"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"time"
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
	now := pcommon.NewTimestampFromTime(time.Now())

	cms.recordCpuStats(now)
	cms.recordMemoryStats(now)
}

func (cms *containerScraper) recordCpuStats(timestamp pcommon.Timestamp) {

	cms.logger.Debug("Collecting container cpu metrics")
	cpuSource := cgroup.NewCPUSource(cgroup.BasePath)
	percentages, err := cpuSource.Collect()
	if err != nil {
		cms.logger.Warn("Failed to collect container cpu metrics", zap.Error(err))
		return
	}

	cms.logger.Debug("Collecting container cpu metrics", zap.Any("percentages", percentages))
	cms.mb.RecordContainerCPUUsageUserDataPoint(timestamp, percentages.User, "0", 0)
	cms.mb.RecordContainerCPUUsageSystemDataPoint(timestamp, percentages.System, "0", 0)
}
func (cms *containerScraper) recordMemoryStats(timestamp pcommon.Timestamp) {
	// capture all the desired memory metrics
	cms.logger.Debug("Collecting container memory metrics")
	memSource := cgroup.NewMemorySource(cgroup.BasePath)
	stats, err := memSource.VirtualMemoryStatWithContext(context.Background())
	if err != nil {
		cms.logger.Warn("Failed to collect container memory metrics", zap.Error(err))
		return
	}
	cms.logger.Debug("Collecting container memory metrics", zap.Any("memory", stats))
	cms.mb.RecordContainerMemoryUsedDataPoint(timestamp, int64(stats.Used))
}
