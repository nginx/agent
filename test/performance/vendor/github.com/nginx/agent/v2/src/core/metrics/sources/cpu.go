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
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"

	"github.com/shirou/gopsutil/v3/cpu"
	log "github.com/sirupsen/logrus"
)

type CPUTimes struct {
	*namedMetric
	isDocker        bool
	cgroupCPUSource *cgroup.CgroupCPU
	logger          *MetricSourceLogger
	// Needed for unit tests
	timesFunc func(bool) ([]cpu.TimesStat, error)
}

type lastTime struct {
	sync.Mutex
	time cpu.TimesStat
}

var lastCPUTime lastTime

func NewCPUTimesSource(namespace string, env core.Environment) *CPUTimes {
	if env.IsContainer() {
		return &CPUTimes{&namedMetric{namespace, CpuGroup}, true, cgroup.NewCgroupCPUSource(cgroup.CgroupBasePath), NewMetricSourceLogger(), nil}
	}
	return &CPUTimes{&namedMetric{namespace, CpuGroup}, false, nil, NewMetricSourceLogger(), cpu.Times}
}

func percentCal(tt float64) func(float64) float64 {
	return func(n float64) float64 {
		if tt == 0.0 {
			return 0.0
		}
		return (n / tt) * 100.00
	}
}

func diffTimeStat(t1, t2 cpu.TimesStat) cpu.TimesStat {
	return cpu.TimesStat{
		CPU:       t1.CPU,
		User:      t2.User - t1.User,
		System:    t2.System - t1.System,
		Idle:      t2.Idle - t1.Idle,
		Nice:      t2.Nice - t1.Nice,
		Iowait:    t2.Iowait - t1.Iowait,
		Irq:       t2.Irq - t1.Irq,
		Softirq:   t2.Softirq - t1.Softirq,
		Steal:     t2.Steal - t1.Steal,
		Guest:     t2.Guest - t1.Guest,
		GuestNice: t2.GuestNice - t1.GuestNice,
	}
}

func (c *CPUTimes) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	var simpleMetrics []*proto.SimpleMetric
	if c.isDocker {
		dockerCpuPercentages, err := c.cgroupCPUSource.Percentages()

		if err != nil {
			// linux impl returns zero length without error
			c.logger.Log(fmt.Sprintf("Failed to get cgroup CPU metrics, %v", err))
			return
		}

		simpleMetrics = c.convertSamplesToSimpleMetrics(map[string]float64{
			"user":   dockerCpuPercentages.User,
			"system": dockerCpuPercentages.System,
		})

		log.Debugf("CPU metrics collected: %v", simpleMetrics)
	} else {
		timesArr, err := c.timesFunc(false)

		if err != nil {
			// linux impl returns zero length without error
			c.logger.Log(fmt.Sprintf("Error occurred getting CPU metrics, %v", err))
			return
		}

		if len(timesArr) != 1 {
			c.logger.Log("Unexpected CPU metrics values")
			return
		}

		currentTime := timesArr[0]

		lastCPUTime.Lock()
		defer lastCPUTime.Unlock()

		lastTime := lastCPUTime.time
		lastCPUTime.time = currentTime

		times := diffTimeStat(lastTime, currentTime)

		tt := times.Total()
		pct := percentCal(tt)
		simpleMetrics = c.convertSamplesToSimpleMetrics(map[string]float64{
			"user":   pct(times.User + times.Nice),
			"system": pct(times.System + times.Irq + times.Softirq),
			"idle":   pct(times.Idle),
			"iowait": pct(times.Iowait),
			"stolen": pct(times.Steal),
		})

		log.Debugf("CPU metrics collected: %v", simpleMetrics)
	}

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}
