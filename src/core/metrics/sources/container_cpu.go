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
	ContainerCpuMetricsWarning = "Unable to collect %s.%s metrics, %v"

	CpuCoresMetricName                = "cores"
	CpuPeriodMetricName               = "period"
	CpuQuotaMetricName                = "quota"
	CpuSharesMetricName               = "shares"
	CpuSetCoresMetricName             = "set.cores"
	CpuThrottlingTimeMetricName       = "throttling.time"
	CpuThrottlingThrottledMetricName  = "throttling.throttled"
	CpuThrottlingPeriodsMetricName    = "throttling.periods"
	CpuThrottlingPercentageMetricName = "throttling.percent"
)

type ContainerCPU struct {
	basePath   string
	isCgroupV2 bool
	logger     *MetricSourceLogger
	*namedMetric
}

func NewContainerCPUSource(namespace string, basePath string) *ContainerCPU {
	log.Trace("Creating new container CPU source")
	return &ContainerCPU{basePath, cgroup.IsCgroupV2(basePath), NewMetricSourceLogger(), &namedMetric{namespace, CpuGroup}}
}

func (c *ContainerCPU) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	log.Trace("Collecting container CPU metrics")
	defer wg.Done()

	containerStats := map[string]float64{}

	if c.isCgroupV2 {
		cpuMax, err := cgroup.ReadSingleValueCgroupFile(path.Join(c.basePath, cgroup.V2CpuMaxFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}
		cpu := strings.Split(cpuMax, " ")

		cpuPeriod, err := strconv.ParseFloat(cpu[1], 64)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		var cpuQuota, cpuCores float64

		// If the cpu quota value is set to max then it will be set to the -1 and the number of
		// cpu cores avaliable to the container is that same as the number of cpu cores
		// of the host system
		if cpu[0] == cgroup.V2DefaultMaxValue {
			cpuQuota = -1
			cpuCores = float64(cgroup.GetNumberOfCores())
		} else {
			cpuQuota, err = strconv.ParseFloat(cpu[0], 64)
			if err != nil {
				c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
				return
			}

			cpuCores = cpuQuota / cpuPeriod
		}

		cpuWeight, err := cgroup.ReadIntegerValueCgroupFile(path.Join(c.basePath, cgroup.V2CpuWeightFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		// This calculation is based on the following documentation:
		// https://github.com/containers/crun/blob/main/crun.1.md#cpu-controller
		cpuShares := 2 + (((cpuWeight - 1) * 262142) / 9999)

		setCores, err := getCPUSetCores(path.Join(c.basePath, cgroup.V2CpusetCpusFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		if setCores == 0 {
			setCores = cpuCores
		}

		cpuThrottlingStats, err := getCPUThrottlingStats(
			path.Join(c.basePath, cgroup.V2CpuStatFile),
			cgroup.V2ThrottlingTimeKey,
			cgroup.V2ThrottlingThrottledKey,
			cgroup.V2ThrottlingPeriodsKey,
		)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		containerStats[CpuCoresMetricName] = cpuCores
		containerStats[CpuPeriodMetricName] = cpuPeriod
		containerStats[CpuQuotaMetricName] = cpuQuota
		containerStats[CpuSharesMetricName] = float64(cpuShares)
		containerStats[CpuSetCoresMetricName] = setCores
		containerStats[CpuThrottlingTimeMetricName] = cpuThrottlingStats[CpuThrottlingTimeMetricName]
		containerStats[CpuThrottlingThrottledMetricName] = cpuThrottlingStats[CpuThrottlingThrottledMetricName]
		containerStats[CpuThrottlingPeriodsMetricName] = cpuThrottlingStats[CpuThrottlingPeriodsMetricName]
		containerStats[CpuThrottlingPercentageMetricName] = cpuThrottlingStats[CpuThrottlingPercentageMetricName]
	} else {
		cpuPeriodString, err := cgroup.ReadSingleValueCgroupFile(path.Join(c.basePath, cgroup.V1CpuPeriodFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		cpuPeriod, err := strconv.ParseFloat(cpuPeriodString, 64)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		cpuQuotaString, err := cgroup.ReadSingleValueCgroupFile(path.Join(c.basePath, cgroup.V1CpuQuotaFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		cpuQuota, err := strconv.ParseFloat(cpuQuotaString, 64)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		var cpuCores float64

		// -1 means that there is no cpu limit set on the container, so the number of
		// cpu cores avaliable to the container is that same as the number of cpu cores
		// of the host system
		if cpuQuotaString == "-1" {
			cpuCores = float64(cgroup.GetNumberOfCores())
		} else {
			cpuCores = cpuQuota / cpuPeriod
		}

		cpuShares, err := cgroup.ReadIntegerValueCgroupFile(path.Join(c.basePath, cgroup.V1CpuSharesFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		setCores, err := getCPUSetCores(path.Join(c.basePath, cgroup.V1CpusetCpusFile))
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		if setCores == 0 {
			setCores = cpuCores
		}

		cpuThrottlingStats, err := getCPUThrottlingStats(
			path.Join(c.basePath, cgroup.V1CpuStatFile),
			cgroup.V1ThrottlingTimeKey,
			cgroup.V1ThrottlingThrottledKey,
			cgroup.V1ThrottlingPeriodsKey,
		)
		if err != nil {
			c.logger.Log(fmt.Sprintf(ContainerCpuMetricsWarning, c.namedMetric.namespace, c.namedMetric.group, err))
			return
		}

		containerStats[CpuCoresMetricName] = cpuCores
		containerStats[CpuPeriodMetricName] = cpuPeriod
		containerStats[CpuQuotaMetricName] = cpuQuota
		containerStats[CpuSharesMetricName] = float64(cpuShares)
		containerStats[CpuSetCoresMetricName] = setCores
		containerStats[CpuThrottlingTimeMetricName] = cpuThrottlingStats[CpuThrottlingTimeMetricName]
		containerStats[CpuThrottlingThrottledMetricName] = cpuThrottlingStats[CpuThrottlingThrottledMetricName]
		containerStats[CpuThrottlingPeriodsMetricName] = cpuThrottlingStats[CpuThrottlingPeriodsMetricName]
		containerStats[CpuThrottlingPercentageMetricName] = cpuThrottlingStats[CpuThrottlingPercentageMetricName]
	}

	simpleMetrics := c.convertSamplesToSimpleMetrics(containerStats)

	log.Debugf("Collected container cpu metrics, %v", simpleMetrics)

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics):
	}
}

func getCPUThrottlingStats(statFile string, time_key string, throttled_key string, periods_key string) (map[string]float64, error) {
	cpuThrottlingStats := map[string]float64{}

	lines, err := cgroup.ReadLines(statFile)
	if err != nil {
		return cpuThrottlingStats, err
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == time_key {
			time, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return cpuThrottlingStats, err
			}

			cpuThrottlingStats[CpuThrottlingTimeMetricName] = time
		}
		if fields[0] == throttled_key {
			throttled, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return cpuThrottlingStats, err
			}

			cpuThrottlingStats[CpuThrottlingThrottledMetricName] = throttled
		}
		if fields[0] == periods_key {
			periods, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return cpuThrottlingStats, err
			}

			cpuThrottlingStats[CpuThrottlingPeriodsMetricName] = periods
		}
	}

	cpuThrottlingStats[CpuThrottlingPercentageMetricName] = (cpuThrottlingStats[CpuThrottlingThrottledMetricName] / cpuThrottlingStats[CpuThrottlingPeriodsMetricName]) * 100

	return cpuThrottlingStats, nil
}

func getCPUSetCores(statFile string) (float64, error) {
	var setCores float64
	setCoresString, err := cgroup.ReadSingleValueCgroupFile(statFile)
	if err != nil {
		return 0, err
	}

	if setCoresString == "" {
		setCores = 0
	} else if strings.Contains(setCoresString, "-") {
		splitSetCoresString := strings.Split(setCoresString, "-")
		if splitSetCoresString[0] == "0" {
			lastCpu, err := strconv.ParseFloat(splitSetCoresString[1], 64)
			if err != nil {
				return 0, err
			}
			setCores = lastCpu + 1
		} else {
			firstCpu, err := strconv.ParseFloat(splitSetCoresString[0], 64)
			if err != nil {
				return 0, err
			}
			lastCpu, err := strconv.ParseFloat(splitSetCoresString[1], 64)
			if err != nil {
				return 0, err
			}

			setCores = lastCpu - firstCpu
		}
	} else if strings.Contains(setCoresString, ",") {
		splitSetCoresString := strings.Split(setCoresString, ",")
		setCores = float64(len(splitSetCoresString))
	} else {
		setCores, err = strconv.ParseFloat(setCoresString, 64)
		if err != nil {
			return 0, err
		}
	}

	return setCores, nil
}
