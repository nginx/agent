/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"math/big"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

type Swap struct {
	*namedMetric
	errorCollectingMetrics error
	statFunc               func() (*mem.SwapMemoryStat, error)
}

func NewSwapSource(namespace string, env core.Environment) *Swap {
	var statFunc = mem.SwapMemory

	if env.IsContainer() {
		cgroupSwapSource := cgroup.NewCgroupSwapSource("/sys/fs/cgroup/")
		statFunc = cgroupSwapSource.SwapMemoryStat
	}

	return &Swap{&namedMetric{namespace, "swap"}, nil, statFunc}
}

func (c *Swap) Name() string {
	return "swap"
}

func (c *Swap) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	swapStats, err := c.statFunc()
	if err != nil {
		c.errorCollectingMetrics = err
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
	c.errorCollectingMetrics = nil

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}

func (c *Swap) ErrorCollectingMetrics() error {
	return c.errorCollectingMetrics
}
