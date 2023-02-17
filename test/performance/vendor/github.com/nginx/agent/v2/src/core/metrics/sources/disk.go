/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/shirou/gopsutil/v3/disk"
)

const MOUNT_POINT = "mount_point"

type Disk struct {
	*namedMetric
	errorCollectingMetrics error
	disks                  []disk.PartitionStat
}

func NewDiskSource(namespace string) *Disk {
	disks, _ := disk.Partitions(false)
	return &Disk{&namedMetric{namespace, "disk"}, nil, disks}
}

func (c *Disk) Name() string {
	return "disk"
}

func (c *Disk) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	for _, part := range c.disks {
		if part.Device == "" || part.Fstype == "" {
			continue
		}
		usage, err := disk.Usage(part.Mountpoint)

		if err != nil {
			c.errorCollectingMetrics = err
			continue
		}

		simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
			"total":  float64(usage.Total),
			"used":   float64(usage.Used),
			"free":   float64(usage.Free),
			"in_use": float64(usage.UsedPercent),
		})

		c.errorCollectingMetrics = nil

		select {
		case <-ctx.Done():
			return
		// mount point is not a common dim
		case m <- metrics.NewStatsEntity([]*proto.Dimension{{Name: MOUNT_POINT, Value: part.Mountpoint}}, simpleMetrics):
		}
	}
}

func (c *Disk) ErrorCollectingMetrics() error {
	return c.errorCollectingMetrics
}
