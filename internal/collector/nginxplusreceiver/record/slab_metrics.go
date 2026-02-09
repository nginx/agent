// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"strconv"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginx/nginx-plus-go-client/v3/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

func RecordSlabPageMetrics(mb *metadata.MetricsBuilder, stats *plusapi.Stats, now pcommon.Timestamp,
	logger *zap.Logger,
) {
	for name, slab := range stats.Slabs {
		mb.RecordNginxSlabPageFreeDataPoint(now, int64(slab.Pages.Free), name)
		mb.RecordNginxSlabPageUsageDataPoint(now, int64(slab.Pages.Used), name)

		for slotName, slot := range slab.Slots {
			slotNumber, err := strconv.ParseInt(slotName, 10, 64)
			if err != nil {
				logger.Warn("Invalid slot name for NGINX Plus slab metrics", zap.Error(err))
			}

			mb.RecordNginxSlabSlotUsageDataPoint(now, int64(slot.Used), slotNumber, name)
			mb.RecordNginxSlabSlotFreeDataPoint(now, int64(slot.Free), slotNumber, name)
			mb.RecordNginxSlabSlotAllocationsDataPoint(
				now,
				int64(slot.Fails),
				metadata.AttributeNginxSlabSlotAllocationResultFAILURE,
				slotNumber,
				name,
			)
			mb.RecordNginxSlabSlotAllocationsDataPoint(
				now,
				int64(slot.Reqs),
				metadata.AttributeNginxSlabSlotAllocationResultSUCCESS,
				slotNumber,
				name,
			)
		}
	}
}
