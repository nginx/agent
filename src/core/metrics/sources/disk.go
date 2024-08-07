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
	log "github.com/sirupsen/logrus"
)

const MOUNT_POINT = "mount_point"

type Disk struct {
	logger *MetricSourceLogger
	*namedMetric
	disks []*proto.DiskPartition
	env   core.Environment
}

func NewDiskSource(namespace string, env core.Environment) *Disk {
	disks, _ := env.Disks()
	return &Disk{NewMetricSourceLogger(), &namedMetric{namespace, "disk"}, disks, env}
}

func (c *Disk) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()
	for _, part := range c.disks {
		if part.Device == "" || part.FsType == "" {
			continue
		}
		usage, err := c.env.DiskUsage(part.MountPoint)
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to get disk metrics for mount point %s, %v", part.MountPoint, err))
			continue
		}

		simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
			"total":  float64(usage.Total),
			"used":   float64(usage.Used),
			"free":   float64(usage.Free),
			"in_use": float64(usage.UsedPercentage),
		})

		log.Debugf("disk metrics collected: %v", len(simpleMetrics))

		select {
		case <-ctx.Done():
			return
		// mount point is not a common dim
		case m <- metrics.NewStatsEntityWrapper([]*proto.Dimension{{Name: MOUNT_POINT, Value: part.MountPoint}}, simpleMetrics, proto.MetricsReport_SYSTEM):
		}
	}
}
