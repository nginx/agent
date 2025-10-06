// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginxinc/nginx-plus-go-client/v2/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func RecordCacheMetrics(mb *metadata.MetricsBuilder, stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, cache := range stats.Caches {
		// Cache Bytes
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Bypass.Bytes),
			metadata.AttributeNginxCacheOutcomeBYPASS,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Expired.Bytes),
			metadata.AttributeNginxCacheOutcomeEXPIRED,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Hit.Bytes),
			metadata.AttributeNginxCacheOutcomeHIT,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Miss.Bytes),
			metadata.AttributeNginxCacheOutcomeMISS,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Revalidated.Bytes),
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Stale.Bytes),
			metadata.AttributeNginxCacheOutcomeSTALE,
			name,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Updating.Bytes),
			metadata.AttributeNginxCacheOutcomeUPDATING,
			name,
		)

		// Cache Memory
		mb.RecordNginxCacheMemoryLimitDataPoint(now, int64(cache.MaxSize), name)
		mb.RecordNginxCacheMemoryUsageDataPoint(now, int64(cache.Size), name)

		// Cache Responses
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Bypass.Responses),
			metadata.AttributeNginxCacheOutcomeBYPASS,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Expired.Responses),
			metadata.AttributeNginxCacheOutcomeEXPIRED,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Hit.Responses),
			metadata.AttributeNginxCacheOutcomeHIT,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Miss.Responses),
			metadata.AttributeNginxCacheOutcomeMISS,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Revalidated.Responses),
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Stale.Responses),
			metadata.AttributeNginxCacheOutcomeSTALE,
			name,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Updating.Responses),
			metadata.AttributeNginxCacheOutcomeUPDATING,
			name,
		)
	}
}
