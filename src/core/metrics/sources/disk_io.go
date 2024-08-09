/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v3/disk"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
)

type DiskIO struct {
	*namedMetric
	diskDevs []string
	// This is for keeping the previous disk io stats.  Need to report the delta.
	// The first level key is the disk device name, and the inside map is the disk
	// io stats for that particular disk device.
	diskIOStats map[string]map[string]float64
	// Needed for unit tests
	diskIOStatsFunc func(ctx context.Context, names ...string) (map[string]disk.IOCountersStat, error)
	init            sync.Once
	env             core.Environment
}

func NewDiskIOSource(namespace string, env core.Environment) *DiskIO {
	return &DiskIO{namedMetric: &namedMetric{namespace, "io"}, env: env, diskIOStatsFunc: disk.IOCountersWithContext}
}

func (dio *DiskIO) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()
	dio.init.Do(func() {
		dio.diskDevs, _ = dio.env.DiskDevices()
		dio.diskIOStats = dio.newDiskIOCounters(ctx, dio.diskDevs)
	})

	// retrieve the current disk IO stats
	currentDiskIOStats := dio.newDiskIOCounters(ctx, dio.diskDevs)

	// calculate the delta between current and previous disk IO stats
	diffDiskIOStats := Delta(currentDiskIOStats, dio.diskIOStats)

	for k, v := range diffDiskIOStats {
		simpleMetrics := dio.convertSamplesToSimpleMetrics(v)
		log.Debugf("disk io metrics collected: %v", len(simpleMetrics))

		select {
		case <-ctx.Done():
			return
		// The psutil is returning the disk IO stats per partition (file_path), not by mount_point,
		// the Controller 3.x was labelling it wrong.  However, changing this on the Analytics side
		// would involve a lot of changes (UI, API, Schema and Ingestion). So we are using mount_point
		// dimension for now.
		case m <- metrics.NewStatsEntityWrapper([]*proto.Dimension{{Name: MOUNT_POINT, Value: k}}, simpleMetrics, proto.MetricsReport_SYSTEM):
		}
	}

	dio.diskIOStats = currentDiskIOStats
}

func isPhysDisk(part string, diskDevs []string) bool {
	for _, dd := range diskDevs {
		if strings.HasPrefix(part, dd) {
			return true
		}
	}
	return false
}

func (dio *DiskIO) newDiskIOCounters(ctx context.Context, diskDevs []string) map[string]map[string]float64 {
	res := make(map[string]map[string]float64)
	diskIOCounters, _ := dio.diskIOStatsFunc(ctx)
	if diskIOCounters == nil {
		log.Debug("Disk IO counters not available")
		return res
	}

	for k, v := range diskIOCounters {
		if !isPhysDisk(k, diskDevs) {
			continue
		}
		res[k] = map[string]float64{
			"iops_w": float64(v.WriteCount),
			"kbs_w":  float64(v.WriteBytes) / 1000,
			"iops_r": float64(v.ReadCount),
			"kbs_r":  float64(v.ReadBytes) / 1000,
			"wait_w": float64(v.WriteTime),
			"wait_r": float64(v.ReadTime),
		}

	}
	return res
}
