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
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	log "github.com/sirupsen/logrus"
)

const (
	ContainerMemoryMetricsWarning = "Unable to collect %s.%s metrics, %v"

	OutOfMemoryMetricName     = "oom"
	OutOfMemoryKillMetricName = "oom.kill"
)

type ContainerMemory struct {
	basePath   string
	isCgroupV2 bool
	logger     *MetricSourceLogger
	*namedMetric
}

func NewContainerMemorySource(namespace string, basePath string) *ContainerMemory {
	log.Trace("Creating new container memory source")
	return &ContainerMemory{basePath, cgroup.IsCgroupV2(basePath), NewMetricSourceLogger(), &namedMetric{namespace, MemoryGroup}}
}

func (c *ContainerMemory) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	log.Trace("Collecting container memory metrics")
	defer wg.Done()

	var containerStats map[string]float64

	if c.isCgroupV2 {
		cpuThrottlingStats, err := getMemOOMStats(path.Join(c.basePath, cgroup.V2MemEventsFile), cgroup.V2OutOfMemoryKey, cgroup.V2OutOfMemoryKillKey)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerMemoryMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		containerStats = cpuThrottlingStats
	} else {
		cpuThrottlingStats, err := getMemOOMStats(path.Join(c.basePath, cgroup.V1OutOfMemoryControlFile), cgroup.V1OutOfMemoryKey, cgroup.V1OutOfMemoryKillKey)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerMemoryMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		containerStats = cpuThrottlingStats
	}

	simpleMetrics := c.convertSamplesToSimpleMetrics(containerStats)

	log.Debugf("Collected container memory metrics, %v", simpleMetrics)

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}

func getMemOOMStats(statFile string, oom_key string, kill_key string) (map[string]float64, error) {
	memOOMStats := map[string]float64{}

	lines, err := cgroup.ReadLines(statFile)
	if err != nil {
		return memOOMStats, err
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == oom_key {
			oom, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return memOOMStats, err
			}

			memOOMStats[OutOfMemoryMetricName] = oom
		}
		if fields[0] == kill_key {
			kill, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return memOOMStats, err
			}

			memOOMStats[OutOfMemoryKillMetricName] = kill
		}
	}

	return memOOMStats, nil
}
