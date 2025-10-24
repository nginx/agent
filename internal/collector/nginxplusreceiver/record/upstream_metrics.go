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
		mb.RecordNginxHTTPUpstreamKeepaliveCountDataPoint(now, int64(upstream.Keepalive), upstream.Zone, name)

		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxIoDirectionReceive,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerConnectionCountDataPoint(
				now,
				int64(peer.Active),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerFailsDataPoint(
				now,
				int64(peer.Fails),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerHeaderTimeDataPoint(
				now,
				int64(peer.HeaderTime),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerRequestsDataPoint(
				now,
				int64(peer.Requests),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Total),
				0,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses1xx),
				metadata.AttributeNginxStatusRange1xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses2xx),
				metadata.AttributeNginxStatusRange2xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses3xx),
				metadata.AttributeNginxStatusRange3xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses4xx),
				metadata.AttributeNginxStatusRange4xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses5xx),
				metadata.AttributeNginxStatusRange5xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerUnavailablesDataPoint(
				now,
				int64(peer.Unavail),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				metadata.AttributeNginxPeerStateCHECKING,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				metadata.AttributeNginxPeerStateDOWN,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				metadata.AttributeNginxPeerStateDRAINING,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				metadata.AttributeNginxPeerStateUNHEALTHY,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUp),
				metadata.AttributeNginxPeerStateUP,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			peerStates[peer.State]++
		}

		// Peer Count
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			upstream.Zone,
			name,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			upstream.Zone,
			name,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			upstream.Zone,
			name,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			upstream.Zone,
			name,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			upstream.Zone,
			name,
		)
		mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			upstream.Zone,
			name,
		)

		// Upstream Queue
		mb.RecordNginxHTTPUpstreamQueueLimitDataPoint(now, int64(upstream.Queue.MaxSize), upstream.Zone, name)
		mb.RecordNginxHTTPUpstreamQueueOverflowsDataPoint(now, int64(upstream.Queue.Overflows), upstream.Zone, name)
		mb.RecordNginxHTTPUpstreamQueueUsageDataPoint(now, int64(upstream.Queue.Size), upstream.Zone, name)
		mb.RecordNginxHTTPUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), upstream.Zone, name)
	}
}

//nolint:revive // booleanValue flag is mandatory
func boolToInt64(booleanValue bool) int64 {
	if booleanValue {
		return 1
	}

	return 0
}
