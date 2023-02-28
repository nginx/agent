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
	"os"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

type VirtualMemory struct {
	logger *MetricSourceLogger
	*namedMetric
	statFunc func() (*mem.VirtualMemoryStat, error)
}

func NewVirtualMemorySource(namespace string, env core.Environment) *VirtualMemory {
	var statFunc = mem.VirtualMemory

	if env.IsContainer() {
		cgroupMemSource := cgroup.NewCgroupMemSource(cgroup.CgroupBasePath)
		statFunc = cgroupMemSource.VirtualMemoryStat
	}

	return &VirtualMemory{NewMetricSourceLogger(), &namedMetric{namespace, MemoryGroup}, statFunc}
}

func (c *VirtualMemory) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	memstats, err := c.statFunc()
	if err != nil {
		if e, ok := err.(*os.PathError); ok {
			c.logger.Log(fmt.Sprintf("Unable to collect VirtualMemory metrics because the file %v was not found", e.Path))
			return
		}
		c.logger.Log(fmt.Sprintf("Failed to collect VirtualMemory metrics, %v", err))
		return
	}

	simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
		"total":     float64(memstats.Total),
		"used":      float64(memstats.Total - memstats.Available),
		"used.all":  float64(memstats.Used),
		"cached":    float64(memstats.Cached),
		"buffered":  float64(memstats.Buffers),
		"shared":    float64(memstats.Shared),
		"pct_used":  float64(memstats.UsedPercent),
		"free":      float64(memstats.Free),
		"available": float64(memstats.Available),
	})

	log.Debugf("Memory metrics collected: %v", simpleMetrics)

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}
