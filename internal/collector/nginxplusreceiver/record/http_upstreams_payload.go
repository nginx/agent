// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	httpUpstreamPeerResponseTimeHistName        = "nginx.http.upstream.peer.response_time_hist"
	httpUpstreamPeerResponseTimeHistDescription = "Histogram of upstream response times collected per upstream server."
	httpUpstreamPeerResponseTimeHistUnit        = "ms"
)

// HTTPUpstreamsPayload mirrors the subset of the NGINX Plus API
// /api/<version>/http/upstreams payload needed to capture response_time_hist,
// which nginx-plus-go-client does not currently expose.
type HTTPUpstreamsPayload map[string]HTTPUpstream

// HTTPUpstream mirrors the upstream-level fields used for the histogram attributes.
type HTTPUpstream struct {
	Zone  string     `json:"zone"`
	Peers []HTTPPeer `json:"peers"`
}

// HTTPPeer mirrors the peer-level fields used for the histogram attributes.
type HTTPPeer struct {
	ResponseTimeHist *ResponseTimeHist `json:"response_time_hist"`
	Server           string            `json:"server"`
	Name             string            `json:"name"`
}

// ResponseTimeHist mirrors the per-peer response time histogram.
type ResponseTimeHist struct {
	Buckets map[string]uint64 `json:"buckets"`
	Count   uint64            `json:"count"`
	Sum     uint64            `json:"sum"`
}

// httpUpstreamPeerResponseTimeBucketKeys is the fixed chronological order of the NGINX
// bucket keys.
var httpUpstreamPeerResponseTimeBucketKeys = []string{
	"5", "10", "25", "50", "75", "100", "250", "500",
	"750", "1000", "2500", "5000", "7500", "10000", "inf",
}

// httpUpstreamPeerResponseTimeBounds are the OTel explicit upper bounds in ms.
var httpUpstreamPeerResponseTimeBounds = []float64{
	5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000,
}

// AppendHTTPUpstreamPeerResponseTimeHist adds the nginx.http.upstream.peer.response_time_hist
// histogram data points to the emitted metrics.
func AppendHTTPUpstreamPeerResponseTimeHist(
	metrics pmetric.Metrics,
	payload HTTPUpstreamsPayload,
	startTime, now pcommon.Timestamp,
) {
	if len(payload) == 0 || metrics.ResourceMetrics().Len() == 0 {
		return
	}

	ils := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)
	m := ils.Metrics().AppendEmpty()
	m.SetName(httpUpstreamPeerResponseTimeHistName)
	m.SetDescription(httpUpstreamPeerResponseTimeHistDescription)
	m.SetUnit(httpUpstreamPeerResponseTimeHistUnit)
	m.SetEmptyHistogram()
	m.Histogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	for upstreamName, upstream := range payload {
		for _, peer := range upstream.Peers {
			h := peer.ResponseTimeHist
			if h == nil {
				continue
			}

			bucketCounts := make([]uint64, 0, len(httpUpstreamPeerResponseTimeBucketKeys))
			for _, key := range httpUpstreamPeerResponseTimeBucketKeys {
				bucketCounts = append(bucketCounts, h.Buckets[key])
			}

			dp := m.Histogram().DataPoints().AppendEmpty()
			dp.SetStartTimestamp(startTime)
			dp.SetTimestamp(now)
			dp.SetCount(h.Count)
			dp.SetSum(float64(h.Sum))
			dp.ExplicitBounds().FromRaw(httpUpstreamPeerResponseTimeBounds)
			dp.BucketCounts().FromRaw(bucketCounts)
			dp.Attributes().PutStr("nginx.peer.address", peer.Server)
			dp.Attributes().PutStr("nginx.peer.name", peer.Name)
			dp.Attributes().PutStr("nginx.upstream.name", upstreamName)
			dp.Attributes().PutStr("nginx.zone.name", upstream.Zone)
		}
	}
}
