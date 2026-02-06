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

const (
	// Peer state is one of “up”, “draining”, “down”, “unavail”, “checking”, and “unhealthy”.
	peerStateUp        = "up"
	peerStateDraining  = "draining"
	peerStateDown      = "down"
	peerStateUnavail   = "unavail"
	peerStateChecking  = "checking"
	peerStateUnhealthy = "unhealthy"
)

func RecordHTTPUpstreamPeerMetrics(mb *metadata.MetricsBuilder, stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, upstream := range stats.Upstreams {
		mb.RecordNginxHTTPUpstreamKeepaliveCountDataPoint(now, int64(upstream.Keepalive), name, upstream.Zone)

		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxIoDirectionReceive,
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerConnectionCountDataPoint(
				now,
				int64(peer.Active),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerFailsDataPoint(
				now,
				int64(peer.Fails),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerHeaderTimeDataPoint(
				now,
				int64(peer.HeaderTime),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerRequestsDataPoint(
				now,
				int64(peer.Requests),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Total),
				peer.Server,
				peer.Name,
				0,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses1xx),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxStatusRange1xx,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses2xx),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxStatusRange2xx,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses3xx),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxStatusRange3xx,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses4xx),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxStatusRange4xx,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses5xx),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxStatusRange5xx,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerUnavailablesDataPoint(
				now,
				int64(peer.Unavail),
				peer.Server,
				peer.Name,
				name,
				upstream.Zone,
			)

			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateCHECKING,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateDOWN,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateDRAINING,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUNHEALTHY,
				name,
				upstream.Zone,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUp),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUP,
				name,
				upstream.Zone,
			)

			peerStates[peer.State]++
		}

		// Peer Count
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			name,
			upstream.Zone,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			name,
			upstream.Zone,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			name,
			upstream.Zone,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			name,
			upstream.Zone,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			name,
			upstream.Zone,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			name,
			upstream.Zone,
		)

		// Upstream Queue
		mb.RecordNginxHTTPUpstreamQueueLimitDataPoint(now, int64(upstream.Queue.MaxSize), name, upstream.Zone)
		mb.RecordNginxHTTPUpstreamQueueOverflowsDataPoint(now, int64(upstream.Queue.Overflows), name, upstream.Zone)
		mb.RecordNginxHTTPUpstreamQueueUsageDataPoint(now, int64(upstream.Queue.Size), name, upstream.Zone)
		mb.RecordNginxHTTPUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), name, upstream.Zone)
	}
}

//nolint:revive // booleanValue flag is mandatory
func boolToInt64(booleanValue bool) int64 {
	if booleanValue {
		return 1
	}

	return 0
}
