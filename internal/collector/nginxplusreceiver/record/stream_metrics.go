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
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			// Connection
			mb.RecordNginxStreamUpstreamPeerConnectionCountDataPoint(
				now,
				int64(peer.Active),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerConnectionTimeDataPoint(
				now,
				int64(peer.ConnectTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			mb.RecordNginxStreamUpstreamPeerConnectionsDataPoint(
				now,
				int64(peer.Connections),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			// Health
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			// Response
			mb.RecordNginxStreamUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerTtfbTimeDataPoint(
				now,
				int64(peer.FirstByteTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerUnavailablesDataPoint(
				now,
				int64(peer.Unavail),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			// State
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				metadata.AttributeNginxPeerStateCHECKING,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				metadata.AttributeNginxPeerStateDOWN,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				metadata.AttributeNginxPeerStateDRAINING,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				metadata.AttributeNginxPeerStateUNHEALTHY,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUp),
				metadata.AttributeNginxPeerStateUP,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			peerStates[peer.State]++
		}

		// Peer Count
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			upstream.Zone,
			upstreamName,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			upstream.Zone,
			upstreamName,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			upstream.Zone,
			upstreamName,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			upstream.Zone,
			upstreamName,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			upstream.Zone,
			upstreamName,
		)
		mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			upstream.Zone,
			upstreamName,
		)

		mb.RecordNginxStreamUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), upstream.Zone, upstreamName)
	}
}
