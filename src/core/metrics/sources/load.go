/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"fmt"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/shirou/gopsutil/v3/load"
)

type Load struct {
	logger *MetricSourceLogger
	*namedMetric
	avgStatsFunc func() (*load.AvgStat, error)
}

func NewLoadSource(namespace string) *Load {
	return &Load{logger: NewMetricSourceLogger(), namedMetric: &namedMetric{namespace, "load"}, avgStatsFunc: load.Avg}
}

func (c *Load) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	loadStats, err := c.avgStatsFunc()
	if err != nil {
		c.logger.Log(fmt.Sprintf("Failed to collect Load metrics, %v", err))
		return
	}

	simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
		"1":  loadStats.Load1,
		"5":  loadStats.Load5,
		"15": loadStats.Load15,
	})

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}
