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

type HTTPMetrics struct {
	mb                        *metadata.MetricsBuilder
	PreviousHTTPRequestsTotal uint64
}

func NewHTTPMetrics(stats *plusapi.Stats, mb *metadata.MetricsBuilder) *HTTPMetrics {
	return &HTTPMetrics{
		mb:                        mb,
		PreviousHTTPRequestsTotal: stats.HTTPRequests.Total,
	}
}

func (hm *HTTPMetrics) RecordHTTPMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	// Requests
	hm.mb.RecordNginxHTTPRequestsDataPoint(now, int64(stats.HTTPRequests.Total), "", 0)

	// Request Count
	requestsDiff := int64(stats.HTTPRequests.Total) - int64(hm.PreviousHTTPRequestsTotal)
	hm.mb.RecordNginxHTTPRequestCountDataPoint(now, requestsDiff, "", 0)
	hm.PreviousHTTPRequestsTotal = stats.HTTPRequests.Total

	// Connections
	hm.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		int64(stats.Connections.Accepted),
		metadata.AttributeNginxConnectionsOutcomeACCEPTED,
	)
	hm.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		int64(stats.Connections.Dropped),
		metadata.AttributeNginxConnectionsOutcomeDROPPED,
	)
	hm.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		int64(stats.Connections.Active),
		metadata.AttributeNginxConnectionsOutcomeACTIVE,
	)
	hm.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		int64(stats.Connections.Idle),
		metadata.AttributeNginxConnectionsOutcomeIDLE,
	)
}

func (hm *HTTPMetrics) RecordHTTPLimitMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	// Limit Connections
	for name, limitConnection := range stats.HTTPLimitConnections {
		hm.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.Passed),
			metadata.AttributeNginxLimitConnOutcomePASSED,
			name,
		)
		hm.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.Rejected),
			metadata.AttributeNginxLimitConnOutcomeREJECTED,
			name,
		)
		hm.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.RejectedDryRun),
			metadata.AttributeNginxLimitConnOutcomeREJECTEDDRYRUN,
			name,
		)
	}

	// Limit Requests
	for name, limitRequest := range stats.HTTPLimitRequests {
		hm.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Passed),
			metadata.AttributeNginxLimitReqOutcomePASSED,
			name,
		)
		hm.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Rejected),
			metadata.AttributeNginxLimitReqOutcomeREJECTED,
			name,
		)
		hm.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.RejectedDryRun),
			metadata.AttributeNginxLimitReqOutcomeREJECTEDDRYRUN,
			name,
		)
		hm.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Delayed),
			metadata.AttributeNginxLimitReqOutcomeDELAYED,
			name,
		)
		hm.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.DelayedDryRun),
			metadata.AttributeNginxLimitReqOutcomeDELAYEDDRYRUN,
			name,
		)
	}
}
