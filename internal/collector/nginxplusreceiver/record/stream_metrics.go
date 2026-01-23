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

func RecordStreamMetrics(mb *metadata.MetricsBuilder, stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, streamServerZone := range stats.StreamServerZones {
		mb.RecordNginxStreamIoDataPoint(
			now,
			int64(streamServerZone.Received),
			metadata.AttributeNginxIoDirectionReceive,
			name,
		)
		mb.RecordNginxStreamIoDataPoint(
			now,
			int64(streamServerZone.Sent),
			metadata.AttributeNginxIoDirectionTransmit,
			name,
		)
		// Connection
		mb.RecordNginxStreamConnectionAcceptedDataPoint(now, int64(streamServerZone.Connections), name)
		mb.RecordNginxStreamConnectionDiscardedDataPoint(now, int64(streamServerZone.Discarded), name)
		mb.RecordNginxStreamConnectionProcessingCountDataPoint(now, int64(streamServerZone.Processing), name)

		// Stream
		mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions2xx),
			metadata.AttributeNginxStatusRange2xx,
			name,
		)
		mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions4xx),
			metadata.AttributeNginxStatusRange4xx,
			name,
		)
		mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions5xx),
			metadata.AttributeNginxStatusRange5xx,
			name,
		)
		mb.RecordNginxStreamSessionStatusDataPoint(now, int64(streamServerZone.Sessions.Total), 0, name)
	}

	// Stream Upstreams
	for upstreamName, upstream := range stats.StreamUpstreams {
		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			mb.RecordNginxStreamUpstreamPeerIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxIoDirectionReceive,
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			// Connection
			mb.RecordNginxStreamUpstreamPeerConnectionCountDataPoint(
				now,
				int64(peer.Active),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerConnectionTimeDataPoint(
				now,
				int64(peer.ConnectTime),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)

			mb.RecordNginxStreamUpstreamPeerConnectionsDataPoint(
				now,
				int64(peer.Connections),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)

			// Health
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)

			// Response
			mb.RecordNginxStreamUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerTtfbTimeDataPoint(
				now,
				int64(peer.FirstByteTime),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerUnavailablesDataPoint(
				now,
				int64(peer.Unavail),
				peer.Server,
				peer.Name,
				upstreamName,
				upstream.Zone,
			)

			// State
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateCHECKING,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateDOWN,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateDRAINING,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUNHEALTHY,
				upstreamName,
				upstream.Zone,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUp),
				peer.Server,
				peer.Name,
				metadata.AttributeNginxPeerStateUP,
				upstreamName,
				upstream.Zone,
			)

			peerStates[peer.State]++
		}

		// Peer Count
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			upstreamName,
			upstream.Zone,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			upstreamName,
			upstream.Zone,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			upstreamName,
			upstream.Zone,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			upstreamName,
			upstream.Zone,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			upstreamName,
			upstream.Zone,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			upstreamName,
			upstream.Zone,
		)

		mb.RecordNginxStreamUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), upstreamName, upstream.Zone)
	}
}
