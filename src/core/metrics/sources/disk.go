package sources

import (
	"context"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/shirou/gopsutil/v3/disk"
	log "github.com/sirupsen/logrus"
)

const MOUNT_POINT = "mount_point"

type Disk struct {
	*namedMetric
	disks []disk.PartitionStat
}

func NewDiskSource(namespace string) *Disk {
	disks, _ := disk.Partitions(false)
	return &Disk{&namedMetric{namespace, "disk"}, disks}
}

func (c *Disk) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	for _, part := range c.disks {
		if part.Device == "" || part.Fstype == "" {
			continue
		}
		usage, err := disk.Usage(part.Mountpoint)

		if err != nil {
			log.Errorf("Failed to get disk metrics %v", err)
			continue
		}

		simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
			"total":  float64(usage.Total),
			"used":   float64(usage.Used),
			"free":   float64(usage.Free),
			"in_use": float64(usage.UsedPercent),
		})

		select {
		case <-ctx.Done():
			return
		// mount point is not a common dim
		case m <- metrics.NewStatsEntity([]*proto.Dimension{{Name: MOUNT_POINT, Value: part.Mountpoint}}, simpleMetrics):
		}
	}
}
