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
	"math/big"
	"os"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

type Swap struct {
	logger *MetricSourceLogger
	*namedMetric
	statFunc func() (*mem.SwapMemoryStat, error)
}

func NewSwapSource(namespace string, env core.Environment) *Swap {
	var statFunc = mem.SwapMemory

	if env.IsContainer() {
		cgroupSwapSource := cgroup.NewCgroupSwapSource("/sys/fs/cgroup/")
		statFunc = cgroupSwapSource.SwapMemoryStat
	}

	// Verify if swap metrics can be collected on startup
	_, err := statFunc()
	if err != nil {
		if e, ok := err.(*os.PathError); ok {
			log.Warnf("Unable to collect Swap metrics because the file %v was not found", e.Path)
		}
		log.Warnf("Unable to collect Swap metrics, %v", err)
	}

	return &Swap{NewMetricSourceLogger(), &namedMetric{namespace, "swap"}, statFunc}
}

func (c *Swap) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	swapStats, err := c.statFunc()
	if err != nil {
		if e, ok := err.(*os.PathError); ok {
			c.logger.Log(fmt.Sprintf("Unable to collect Swap metrics because the file %v was not found", e.Path))
			return
		}
		c.logger.Log(fmt.Sprintf("Unable to collect Swap metrics, %v", err))
		return
	}

	percentageFree, _ := new(big.Float).Sub(new(big.Float).SetFloat64(100.0), new(big.Float).SetFloat64(swapStats.UsedPercent)).Float64()

	simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
		"total":    float64(swapStats.Total),
		"used":     float64(swapStats.Used),
		"free":     float64(swapStats.Free),
		"pct_free": percentageFree,
	})

	log.Debugf("Swap Memory metrics collected: %v", simpleMetrics)

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}
