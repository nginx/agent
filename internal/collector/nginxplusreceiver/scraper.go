// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginxinc/nginx-plus-go-client/client"
)

const (
	plusAPIVersion = 9

	// Peer state is one of “up”, “draining”, “down”, “unavail”, “checking”, and “unhealthy”.
	peerStateUp        = "up"
	peerStateDraining  = "draining"
	peerStateDown      = "down"
	peerStateUnavail   = "unavail"
	peerStateChecking  = "checking"
	peerStateUnhealthy = "unhealthy"
)

type nginxPlusScraper struct {
	plusClient *plusapi.NginxClient

	settings component.TelemetrySettings
	cfg      *Config
	mb       *metadata.MetricsBuilder
}

func newNginxPlusScraper(
	settings receiver.Settings,
	cfg *Config,
) (*nginxPlusScraper, error) {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)

	plusClient, err := plusapi.NewNginxClient(cfg.Endpoint,
		plusapi.WithAPIVersion(plusAPIVersion),
	)
	if err != nil {
		return nil, err
	}

	return &nginxPlusScraper{
		plusClient: plusClient,
		settings:   settings.TelemetrySettings,
		cfg:        cfg,
		mb:         mb,
	}, nil
}

func (nps *nginxPlusScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	stats, err := nps.plusClient.GetStats()
	if err != nil {
		return pmetric.Metrics{}, fmt.Errorf("GET stats: %w", err)
	}

	slog.DebugContext(ctx, "NGINX Plus stats", "stats", stats)

	nps.recordMetrics(ctx, stats)

	return nps.mb.Emit(), nil
}

func (nps *nginxPlusScraper) recordMetrics(ctx context.Context, stats *plusapi.Stats) {
	now := pcommon.NewTimestampFromTime(time.Now())

	// NGINX config reloads
	nps.mb.RecordNginxConfigReloadsDataPoint(now, int64(stats.NginxInfo.Generation))

	// Connections
	nps.mb.RecordNginxHTTPConnDataPoint(
		now,
		int64(stats.Connections.Accepted),
		metadata.AttributeNginxConnOutcomeACCEPTED,
	)
	nps.mb.RecordNginxHTTPConnDataPoint(
		now,
		int64(stats.Connections.Dropped),
		metadata.AttributeNginxConnOutcomeDROPPED,
	)
	nps.mb.RecordNginxHTTPConnCountDataPoint(
		now,
		int64(stats.Connections.Active),
		metadata.AttributeNginxConnOutcomeACTIVE,
	)
	nps.mb.RecordNginxHTTPConnCountDataPoint(now, int64(stats.Connections.Idle), metadata.AttributeNginxConnOutcomeIDLE)

	// HTTP Requests
	nps.mb.RecordNginxHTTPRequestsDataPoint(now, int64(stats.HTTPRequests.Total), "", 0)
	nps.mb.RecordNginxHTTPRequestsCountDataPoint(now, int64(stats.HTTPRequests.Current))

	nps.recordCacheMetrics(stats, now)
	nps.recordHTTPLimitMetrics(stats, now)
	nps.recordLocationZoneMetrics(stats, now)
	nps.recordServerZoneMetrics(stats, now)
	nps.recordHTTPUpstreamPeerMetrics(stats, now)
	nps.recordSlabPageMetrics(ctx, stats, now)
	nps.recordSSLMetrics(now, stats)
	nps.recordStreamMetrics(stats, now)
}

func (nps *nginxPlusScraper) recordStreamMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, streamServerZone := range stats.StreamServerZones {
		nps.mb.RecordNginxStreamByteIoDataPoint(
			now,
			int64(streamServerZone.Received),
			metadata.AttributeNginxByteIoDirectionRX,
			name,
		)
		nps.mb.RecordNginxStreamByteIoDataPoint(
			now,
			int64(streamServerZone.Sent),
			metadata.AttributeNginxByteIoDirectionTX,
			name,
		)
		nps.mb.RecordNginxStreamConnectionAcceptedDataPoint(now, int64(streamServerZone.Connections), name)
		nps.mb.RecordNginxStreamConnectionDiscardedDataPoint(now, int64(streamServerZone.Discarded), name)
		nps.mb.RecordNginxStreamConnectionProcessingCountDataPoint(now, int64(streamServerZone.Processing), name)
		nps.mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions2xx),
			metadata.AttributeNginxStatusRange2xx,
			name,
		)
		nps.mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions4xx),
			metadata.AttributeNginxStatusRange4xx,
			name,
		)
		nps.mb.RecordNginxStreamSessionStatusDataPoint(
			now,
			int64(streamServerZone.Sessions.Sessions5xx),
			metadata.AttributeNginxStatusRange5xx,
			name,
		)
		nps.mb.RecordNginxStreamSessionStatusDataPoint(now, int64(streamServerZone.Sessions.Total), 0, name)
	}

	for upstreamName, upstream := range stats.StreamUpstreams {
		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			nps.mb.RecordNginxStreamUpstreamPeerByteIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxByteIoDirectionRX,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerByteIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxByteIoDirectionTX,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerConnCountDataPoint(
				now,
				int64(peer.Active),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerConnTimeDataPoint(
				now,
				int64(peer.ConnectTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerConnsDataPoint(
				now,
				int64(peer.Connections),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxStreamUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerTtfbTimeDataPoint(
				now,
				int64(peer.FirstByteTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerUnavailableDataPoint(
				now,
				int64(peer.Unavail),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				metadata.AttributeNginxPeerStateCHECKING,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				metadata.AttributeNginxPeerStateDOWN,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				metadata.AttributeNginxPeerStateDRAINING,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				metadata.AttributeNginxPeerStateUNHEALTHY,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerStateDataPoint(
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

		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			upstream.Zone,
			upstreamName,
		)
		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			upstream.Zone,
			upstreamName,
		)
		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			upstream.Zone,
			upstreamName,
		)
		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			upstream.Zone,
			upstreamName,
		)
		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			upstream.Zone,
			upstreamName,
		)
		nps.mb.RecordNginxStreamUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			upstream.Zone,
			upstreamName,
		)

		nps.mb.RecordNginxStreamUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), upstream.Zone, upstreamName)
	}
}

func (nps *nginxPlusScraper) recordSSLMetrics(now pcommon.Timestamp, stats *plusapi.Stats) {
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.HandshakesFailed),
		metadata.AttributeNginxSslStatusFAILED,
		0,
	)
	nps.mb.RecordNginxSslHandshakesDataPoint(now, int64(stats.SSL.Handshakes), 0, 0)
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.SessionReuses),
		metadata.AttributeNginxSslStatusREUSE,
		0,
	)
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonProtocol),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONPROTOCOL,
	)
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.NoCommonCipher),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonNOCOMMONCIPHER,
	)
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.HandshakeTimeout),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonTIMEOUT,
	)
	nps.mb.RecordNginxSslHandshakesDataPoint(
		now,
		int64(stats.SSL.PeerRejectedCert),
		metadata.AttributeNginxSslStatusFAILED,
		metadata.AttributeNginxSslHandshakeReasonCERTREJECTED,
	)

	nps.mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.NoCert),
		metadata.AttributeNginxSslVerifyFailureReasonNOCERT,
	)
	nps.mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.ExpiredCert),
		metadata.AttributeNginxSslVerifyFailureReasonEXPIREDCERT,
	)
	nps.mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.RevokedCert),
		metadata.AttributeNginxSslVerifyFailureReasonREVOKEDCERT,
	)
	nps.mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.HostnameMismatch),
		metadata.AttributeNginxSslVerifyFailureReasonHOSTNAMEMISMATCH,
	)
	nps.mb.RecordNginxSslCertificateVerifyFailuresDataPoint(
		now,
		int64(stats.SSL.VerifyFailures.Other),
		metadata.AttributeNginxSslVerifyFailureReasonOTHER,
	)
}

func (nps *nginxPlusScraper) recordSlabPageMetrics(ctx context.Context, stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, slab := range stats.Slabs {
		nps.mb.RecordNginxSlabPageFreeDataPoint(now, int64(slab.Pages.Free), name)
		nps.mb.RecordNginxSlabPageUsageDataPoint(now, int64(slab.Pages.Used), name)

		for slotName, slot := range slab.Slots {
			slotNumber, err := strconv.ParseInt(slotName, 10, 64)
			if err != nil {
				slog.WarnContext(ctx, "Invalid slot name for NGINX Plus slab metrics", "error", err)
			}

			nps.mb.RecordNginxSlabSlotUsageDataPoint(now, int64(slot.Used), slotNumber, name)
			nps.mb.RecordNginxSlabSlotFreeDataPoint(now, int64(slot.Free), slotNumber, name)
			nps.mb.RecordNginxSlabSlotAllocationsDataPoint(
				now,
				int64(slot.Fails),
				slotNumber,
				metadata.AttributeNginxSlabSlotAllocationResultFAILURE,
				name,
			)
			nps.mb.RecordNginxSlabSlotAllocationsDataPoint(
				now,
				int64(slot.Reqs),
				slotNumber,
				metadata.AttributeNginxSlabSlotAllocationResultSUCCESS,
				name,
			)
		}
	}
}

func (nps *nginxPlusScraper) recordHTTPUpstreamPeerMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, upstream := range stats.Upstreams {
		nps.mb.RecordNginxHTTPUpstreamKeepaliveCountDataPoint(now, int64(upstream.Keepalive), upstream.Zone, name)

		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			nps.mb.RecordNginxHTTPUpstreamPeerByteIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxByteIoDirectionRX,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerByteIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxByteIoDirectionTX,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerConnCountDataPoint(
				now,
				int64(peer.Active),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerFailsDataPoint(
				now,
				int64(peer.Fails),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerHeaderTimeDataPoint(
				now,
				int64(peer.HeaderTime),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Checks),
				0,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Fails),
				metadata.AttributeNginxHealthCheckFAIL,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerHealthChecksDataPoint(
				now,
				int64(peer.HealthChecks.Unhealthy),
				metadata.AttributeNginxHealthCheckUNHEALTHY,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerRequestsDataPoint(
				now,
				int64(peer.Requests),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponseTimeDataPoint(
				now,
				int64(peer.ResponseTime),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Total),
				0,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses1xx),
				metadata.AttributeNginxStatusRange1xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses2xx),
				metadata.AttributeNginxStatusRange2xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses3xx),
				metadata.AttributeNginxStatusRange3xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses4xx),
				metadata.AttributeNginxStatusRange4xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerResponsesDataPoint(
				now,
				int64(peer.Responses.Responses5xx),
				metadata.AttributeNginxStatusRange5xx,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerUnavailablesDataPoint(
				now,
				int64(peer.Unavail),
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateChecking),
				metadata.AttributeNginxPeerStateCHECKING,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDown),
				metadata.AttributeNginxPeerStateDOWN,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateDraining),
				metadata.AttributeNginxPeerStateDRAINING,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnavail),
				metadata.AttributeNginxPeerStateUNAVAILABLE,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
				now,
				boolToInt64(peer.State == peerStateUnhealthy),
				metadata.AttributeNginxPeerStateUNHEALTHY,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerStateDataPoint(
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

		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateChecking]),
			metadata.AttributeNginxPeerStateCHECKING,
			upstream.Zone,
			name,
		)
		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDown]),
			metadata.AttributeNginxPeerStateDOWN,
			upstream.Zone,
			name,
		)
		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateDraining]),
			metadata.AttributeNginxPeerStateDRAINING,
			upstream.Zone,
			name,
		)
		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnavail]),
			metadata.AttributeNginxPeerStateUNAVAILABLE,
			upstream.Zone,
			name,
		)
		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUnhealthy]),
			metadata.AttributeNginxPeerStateUNHEALTHY,
			upstream.Zone,
			name,
		)
		nps.mb.RecordNginxHTTPUpstreamPeerCountDataPoint(
			now,
			int64(peerStates[peerStateUp]),
			metadata.AttributeNginxPeerStateUP,
			upstream.Zone,
			name,
		)

		nps.mb.RecordNginxHTTPUpstreamQueueLimitDataPoint(now, int64(upstream.Queue.MaxSize), upstream.Zone, name)
		nps.mb.RecordNginxHTTPUpstreamQueueOverflowsDataPoint(now, int64(upstream.Queue.Overflows), upstream.Zone, name)
		nps.mb.RecordNginxHTTPUpstreamQueueUsageDataPoint(now, int64(upstream.Queue.Size), upstream.Zone, name)
		nps.mb.RecordNginxHTTPUpstreamZombieCountDataPoint(now, int64(upstream.Zombies), upstream.Zone, name)
	}
}

func (nps *nginxPlusScraper) recordServerZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for szName, sz := range stats.ServerZones {
		nps.mb.RecordNginxHTTPRequestByteIoDataPoint(
			now,
			int64(sz.Received),
			metadata.AttributeNginxByteIoDirectionRX,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		nps.mb.RecordNginxHTTPRequestByteIoDataPoint(
			now,
			int64(sz.Sent),
			metadata.AttributeNginxByteIoDirectionTX,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestsDataPoint(now, int64(sz.Requests), szName, metadata.AttributeNginxZoneTypeSERVER)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses1xx),
			metadata.AttributeNginxStatusRange1xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses2xx),
			metadata.AttributeNginxStatusRange2xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses3xx),
			metadata.AttributeNginxStatusRange3xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses4xx),
			metadata.AttributeNginxStatusRange4xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses5xx),
			metadata.AttributeNginxStatusRange5xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, int64(sz.Discarded),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestProcessingCountDataPoint(now, int64(sz.Processing),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
	}
}

func (nps *nginxPlusScraper) recordLocationZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for lzName, lz := range stats.LocationZones {
		nps.mb.RecordNginxHTTPRequestByteIoDataPoint(
			now,
			lz.Received,
			metadata.AttributeNginxByteIoDirectionRX,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPRequestByteIoDataPoint(
			now,
			lz.Sent,
			metadata.AttributeNginxByteIoDirectionTX,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPRequestsDataPoint(
			now,
			lz.Requests,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses1xx),
			metadata.AttributeNginxStatusRange1xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses2xx),
			metadata.AttributeNginxStatusRange2xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses3xx),
			metadata.AttributeNginxStatusRange3xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses4xx),
			metadata.AttributeNginxStatusRange4xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses5xx),
			metadata.AttributeNginxStatusRange5xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, lz.Discarded,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
	}
}

func (nps *nginxPlusScraper) recordHTTPLimitMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, limitConnection := range stats.HTTPLimitConnections {
		nps.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.Passed),
			metadata.AttributeNginxLimitConnOutcomePASSED,
			name,
		)
		nps.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.Rejected),
			metadata.AttributeNginxLimitConnOutcomeREJECTED,
			name,
		)
		nps.mb.RecordNginxHTTPLimitConnRequestsDataPoint(
			now,
			int64(limitConnection.RejectedDryRun),
			metadata.AttributeNginxLimitConnOutcomeREJECTEDDRYRUN,
			name,
		)
	}

	for name, limitRequest := range stats.HTTPLimitRequests {
		nps.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Passed),
			metadata.AttributeNginxLimitReqOutcomePASSED,
			name,
		)
		nps.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Rejected),
			metadata.AttributeNginxLimitReqOutcomeREJECTED,
			name,
		)
		nps.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.RejectedDryRun),
			metadata.AttributeNginxLimitReqOutcomeREJECTEDDRYRUN,
			name,
		)
		nps.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.Delayed),
			metadata.AttributeNginxLimitReqOutcomeDELAYED,
			name,
		)
		nps.mb.RecordNginxHTTPLimitReqRequestsDataPoint(
			now,
			int64(limitRequest.DelayedDryRun),
			metadata.AttributeNginxLimitReqOutcomeDELAYEDDRYRUN,
			name,
		)
	}
}

func (nps *nginxPlusScraper) recordCacheMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, cache := range stats.Caches {
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Bypass.Bytes),
			metadata.AttributeNginxCacheOutcomeBYPASS,
			name,
		)
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Expired.Bytes),
			metadata.AttributeNginxCacheOutcomeEXPIRED,
			name,
		)
		nps.mb.RecordNginxCacheBytesDataPoint(now, int64(cache.Hit.Bytes), metadata.AttributeNginxCacheOutcomeHIT, name)
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Miss.Bytes),
			metadata.AttributeNginxCacheOutcomeMISS,
			name,
		)
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Revalidated.Bytes),
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
			name,
		)
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Stale.Bytes),
			metadata.AttributeNginxCacheOutcomeSTALE,
			name,
		)
		nps.mb.RecordNginxCacheBytesDataPoint(
			now,
			int64(cache.Updating.Bytes),
			metadata.AttributeNginxCacheOutcomeUPDATING,
			name,
		)

		nps.mb.RecordNginxCacheMemoryLimitDataPoint(now, int64(cache.MaxSize), name)
		nps.mb.RecordNginxCacheMemoryUsageDataPoint(now, int64(cache.Size), name)

		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Bypass.Responses),
			metadata.AttributeNginxCacheOutcomeBYPASS,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Expired.Responses),
			metadata.AttributeNginxCacheOutcomeEXPIRED,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Hit.Responses),
			metadata.AttributeNginxCacheOutcomeHIT,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Miss.Responses),
			metadata.AttributeNginxCacheOutcomeMISS,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Revalidated.Responses),
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Stale.Responses),
			metadata.AttributeNginxCacheOutcomeSTALE,
			name,
		)
		nps.mb.RecordNginxCacheResponsesDataPoint(
			now,
			int64(cache.Updating.Responses),
			metadata.AttributeNginxCacheOutcomeUPDATING,
			name,
		)
	}
}

func (nps *nginxPlusScraper) Shutdown(ctx context.Context) error {
	return nil
}

// nolint: revive
func boolToInt64(booleanValue bool) int64 {
	if booleanValue {
		return 1
	}

	return 0
}
