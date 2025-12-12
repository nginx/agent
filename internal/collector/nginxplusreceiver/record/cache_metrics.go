// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginx/nginx-plus-go-client/v3/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func RecordCacheMetrics(mb *metadata.MetricsBuilder, stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, cache := range stats.Caches {
		// Cache Bytes
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Bypass.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeBYPASS,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Expired.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeEXPIRED,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Hit.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeHIT,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Miss.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeMISS,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Revalidated.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Stale.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeSTALE,
		)
		mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Updating.Bytes),
			name,
			metadata.AttributeNginxCacheOutcomeUPDATING,
		)

		// Cache Memory
		mb.RecordNginxCacheMemoryLimitDataPoint(now, int64(cache.MaxSize), name)
		mb.RecordNginxCacheMemoryUsageDataPoint(now, int64(cache.Size), name)

		// Cache Responses
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Bypass.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeBYPASS,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Expired.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeEXPIRED,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Hit.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeHIT,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Miss.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeMISS,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Revalidated.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Stale.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeSTALE,
		)
		mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Updating.Responses),
			name,
			metadata.AttributeNginxCacheOutcomeUPDATING,
		)
	}
}
