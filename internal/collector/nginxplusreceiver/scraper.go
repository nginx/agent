// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginxinc/nginx-plus-go-client/v2/client"
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

type NginxPlusScraper struct {
	previousServerZoneResponses   map[string]ResponseStatuses
	previousLocationZoneResponses map[string]ResponseStatuses
	plusClient                    *plusapi.NginxClient
	cfg                           *Config
	mb                            *metadata.MetricsBuilder
	rb                            *metadata.ResourceBuilder
	logger                        *zap.Logger
	settings                      receiver.Settings
	init                          sync.Once
	previousHTTPRequestsTotal     uint64
}

type ResponseStatuses struct {
	oneHundredStatusRange   int64
	twoHundredStatusRange   int64
	threeHundredStatusRange int64
	fourHundredStatusRange  int64
	fiveHundredStatusRange  int64
}

func newNginxPlusScraper(
	settings receiver.Settings,
	cfg *Config,
) *NginxPlusScraper {
	logger := settings.Logger
	logger.Info("Creating NGINX Plus scraper")
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &NginxPlusScraper{
		settings: settings,
		cfg:      cfg,
		mb:       mb,
		rb:       rb,
		logger:   settings.Logger,
	}
}

func (nps *NginxPlusScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (nps *NginxPlusScraper) Start(_ context.Context, _ component.Host) error {
	endpoint := strings.TrimPrefix(nps.cfg.APIDetails.URL, "unix:")
	httpClient := http.DefaultClient
	caCertLocation := nps.cfg.APIDetails.Ca
	if caCertLocation != "" {
		nps.logger.Debug("Reading CA certificate", zap.Any("file_path", caCertLocation))
		caCert, err := os.ReadFile(caCertLocation)
		if err != nil {
			nps.logger.Error("Error starting NGINX stub status scraper. "+
				"Failed to read CA certificate", zap.Error(err))

			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    caCertPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}
	}
	httpClient.Timeout = nps.cfg.ClientConfig.Timeout

	if strings.HasPrefix(nps.cfg.APIDetails.Listen, "unix:") {
		httpClient = socketClient(strings.TrimPrefix(nps.cfg.APIDetails.Listen, "unix:"))
	}

	plusClient, err := plusapi.NewNginxClient(endpoint,
		plusapi.WithMaxAPIVersion(), plusapi.WithHTTPClient(httpClient),
	)
	nps.plusClient = plusClient
	if err != nil {
		return err
	}

	return nil
}

func (nps *NginxPlusScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// nps.init.Do is ran only once, it is only ran the first time scrape is called to set the previous responses
	// metric value
	nps.init.Do(func() {
		stats, err := nps.plusClient.GetStats(ctx)
		if err != nil {
			nps.logger.Error("Failed to get stats from plus API", zap.Error(err))
			return
		}

		nps.previousHTTPRequestsTotal = stats.HTTPRequests.Total
		nps.createPreviousServerZoneResponses(stats)
		nps.createPreviousLocationZoneResponses(stats)
	})

	stats, err := nps.plusClient.GetStats(ctx)
	if err != nil {
		return pmetric.Metrics{}, fmt.Errorf("failed to get stats from plus API: %w", err)
	}

	nps.rb.SetInstanceID(nps.settings.ID.Name())
	nps.rb.SetInstanceType("nginxplus")
	nps.logger.Debug("NGINX Plus resource info", zap.Any("resource", nps.rb))

	nps.logger.Debug("NGINX Plus stats", zap.Any("stats", stats))
	nps.recordMetrics(stats)

	return nps.mb.Emit(metadata.WithResource(nps.rb.Emit())), nil
}

func (nps *NginxPlusScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (nps *NginxPlusScraper) createPreviousLocationZoneResponses(stats *plusapi.Stats) {
	previousLocationZoneResponses := make(map[string]ResponseStatuses)
	for lzName, lz := range stats.LocationZones {
		respStatus := ResponseStatuses{
			oneHundredStatusRange:   int64(lz.Responses.Responses1xx),
			twoHundredStatusRange:   int64(lz.Responses.Responses2xx),
			threeHundredStatusRange: int64(lz.Responses.Responses3xx),
			fourHundredStatusRange:  int64(lz.Responses.Responses4xx),
			fiveHundredStatusRange:  int64(lz.Responses.Responses5xx),
		}

		previousLocationZoneResponses[lzName] = respStatus
	}

	nps.previousLocationZoneResponses = previousLocationZoneResponses
}

func (nps *NginxPlusScraper) createPreviousServerZoneResponses(stats *plusapi.Stats) {
	previousServerZoneResponses := make(map[string]ResponseStatuses)
	for szName, sz := range stats.ServerZones {
		respStatus := ResponseStatuses{
			oneHundredStatusRange:   int64(sz.Responses.Responses1xx),
			twoHundredStatusRange:   int64(sz.Responses.Responses2xx),
			threeHundredStatusRange: int64(sz.Responses.Responses3xx),
			fourHundredStatusRange:  int64(sz.Responses.Responses4xx),
			fiveHundredStatusRange:  int64(sz.Responses.Responses5xx),
		}

		previousServerZoneResponses[szName] = respStatus
	}

	nps.previousServerZoneResponses = previousServerZoneResponses
}

func (nps *NginxPlusScraper) recordMetrics(stats *plusapi.Stats) {
	now := pcommon.NewTimestampFromTime(time.Now())

	// NGINX config reloads
	nps.mb.RecordNginxConfigReloadsDataPoint(now, int64(stats.NginxInfo.Generation))

	// Connections
	nps.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		int64(stats.Connections.Accepted),
		metadata.AttributeNginxConnectionsOutcomeACCEPTED,
	)
	nps.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		int64(stats.Connections.Dropped),
		metadata.AttributeNginxConnectionsOutcomeDROPPED,
	)
	nps.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		int64(stats.Connections.Active),
		metadata.AttributeNginxConnectionsOutcomeACTIVE,
	)
	nps.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		int64(stats.Connections.Idle),
		metadata.AttributeNginxConnectionsOutcomeIDLE,
	)

	// HTTP Requests
	nps.mb.RecordNginxHTTPRequestsDataPoint(now, int64(stats.HTTPRequests.Total), "", 0)

	requestsDiff := int64(stats.HTTPRequests.Total) - int64(nps.previousHTTPRequestsTotal)
	nps.mb.RecordNginxHTTPRequestCountDataPoint(now, requestsDiff)
	nps.previousHTTPRequestsTotal = stats.HTTPRequests.Total

	nps.recordCacheMetrics(stats, now)
	nps.recordHTTPLimitMetrics(stats, now)
	nps.recordLocationZoneMetrics(stats, now)
	nps.recordServerZoneMetrics(stats, now)
	nps.recordHTTPUpstreamPeerMetrics(stats, now)
	nps.recordSlabPageMetrics(stats, now)
	nps.recordSSLMetrics(now, stats)
	nps.recordStreamMetrics(stats, now)
}

func (nps *NginxPlusScraper) recordStreamMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, streamServerZone := range stats.StreamServerZones {
		nps.mb.RecordNginxStreamIoDataPoint(
			now,
			int64(streamServerZone.Received),
			metadata.AttributeNginxIoDirectionReceive,
			name,
		)
		nps.mb.RecordNginxStreamIoDataPoint(
			now,
			int64(streamServerZone.Sent),
			metadata.AttributeNginxIoDirectionTransmit,
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
			nps.mb.RecordNginxStreamUpstreamPeerIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxIoDirectionReceive,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerConnectionCountDataPoint(
				now,
				int64(peer.Active),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxStreamUpstreamPeerConnectionTimeDataPoint(
				now,
				int64(peer.ConnectTime),
				upstream.Zone,
				upstreamName,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxStreamUpstreamPeerConnectionsDataPoint(
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
			nps.mb.RecordNginxStreamUpstreamPeerUnavailablesDataPoint(
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

func (nps *NginxPlusScraper) recordSSLMetrics(now pcommon.Timestamp, stats *plusapi.Stats) {
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

func (nps *NginxPlusScraper) recordSlabPageMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, slab := range stats.Slabs {
		nps.mb.RecordNginxSlabPageFreeDataPoint(now, int64(slab.Pages.Free), name)
		nps.mb.RecordNginxSlabPageUsageDataPoint(now, int64(slab.Pages.Used), name)

		for slotName, slot := range slab.Slots {
			slotNumber, err := strconv.ParseInt(slotName, 10, 64)
			if err != nil {
				nps.logger.Warn("Invalid slot name for NGINX Plus slab metrics", zap.Error(err))
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

func (nps *NginxPlusScraper) recordHTTPUpstreamPeerMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, upstream := range stats.Upstreams {
		nps.mb.RecordNginxHTTPUpstreamKeepaliveCountDataPoint(now, int64(upstream.Keepalive), upstream.Zone, name)

		peerStates := make(map[string]int)

		for _, peer := range upstream.Peers {
			nps.mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Received),
				metadata.AttributeNginxIoDirectionReceive,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)
			nps.mb.RecordNginxHTTPUpstreamPeerIoDataPoint(
				now,
				int64(peer.Sent),
				metadata.AttributeNginxIoDirectionTransmit,
				upstream.Zone,
				name,
				peer.Server,
				peer.Name,
			)

			nps.mb.RecordNginxHTTPUpstreamPeerConnectionCountDataPoint(
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

func (nps *NginxPlusScraper) recordServerZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for szName, sz := range stats.ServerZones {
		nps.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			int64(sz.Received),
			metadata.AttributeNginxIoDirectionReceive,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		nps.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			int64(sz.Sent),
			metadata.AttributeNginxIoDirectionTransmit,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestsDataPoint(now, int64(sz.Requests), szName, metadata.AttributeNginxZoneTypeSERVER)

		nps.recordServerZoneHTTPMetrics(sz, szName, now)

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

// Duplicate of recordLocationZoneHTTPMetrics but same function can not be used due to plusapi.ServerZone
// nolint: dupl
func (nps *NginxPlusScraper) recordServerZoneHTTPMetrics(sz plusapi.ServerZone, szName string, now pcommon.Timestamp) {
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

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses1xx)-nps.previousServerZoneResponses[szName].oneHundredStatusRange,
		metadata.AttributeNginxStatusRange1xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses2xx)-nps.previousServerZoneResponses[szName].twoHundredStatusRange,
		metadata.AttributeNginxStatusRange2xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses3xx)-nps.previousServerZoneResponses[szName].threeHundredStatusRange,
		metadata.AttributeNginxStatusRange3xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses4xx)-nps.previousServerZoneResponses[szName].fourHundredStatusRange,
		metadata.AttributeNginxStatusRange4xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses5xx)-nps.previousServerZoneResponses[szName].fiveHundredStatusRange,
		metadata.AttributeNginxStatusRange5xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	respStatus := ResponseStatuses{
		oneHundredStatusRange:   int64(sz.Responses.Responses1xx),
		twoHundredStatusRange:   int64(sz.Responses.Responses2xx),
		threeHundredStatusRange: int64(sz.Responses.Responses3xx),
		fourHundredStatusRange:  int64(sz.Responses.Responses4xx),
		fiveHundredStatusRange:  int64(sz.Responses.Responses5xx),
	}

	nps.previousServerZoneResponses[szName] = respStatus
}

func (nps *NginxPlusScraper) recordLocationZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for lzName, lz := range stats.LocationZones {
		nps.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			lz.Received,
			metadata.AttributeNginxIoDirectionReceive,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			lz.Sent,
			metadata.AttributeNginxIoDirectionTransmit,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPRequestsDataPoint(
			now,
			lz.Requests,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.recordLocationZoneHTTPMetrics(lz, lzName, now)

		nps.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, lz.Discarded,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
	}
}

// Duplicate of recordServerZoneHTTPMetrics but same function can not be used due to plusapi.LocationZone
// nolint: dupl
func (nps *NginxPlusScraper) recordLocationZoneHTTPMetrics(lz plusapi.LocationZone,
	lzName string, now pcommon.Timestamp,
) {
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

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses1xx)-nps.previousLocationZoneResponses[lzName].oneHundredStatusRange,
		metadata.AttributeNginxStatusRange1xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses2xx)-nps.previousLocationZoneResponses[lzName].twoHundredStatusRange,
		metadata.AttributeNginxStatusRange2xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses3xx)-nps.previousLocationZoneResponses[lzName].threeHundredStatusRange,
		metadata.AttributeNginxStatusRange3xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses4xx)-nps.previousLocationZoneResponses[lzName].fourHundredStatusRange,
		metadata.AttributeNginxStatusRange4xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	nps.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses5xx)-nps.previousLocationZoneResponses[lzName].fiveHundredStatusRange,
		metadata.AttributeNginxStatusRange5xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	respStatus := ResponseStatuses{
		oneHundredStatusRange:   int64(lz.Responses.Responses1xx),
		twoHundredStatusRange:   int64(lz.Responses.Responses2xx),
		threeHundredStatusRange: int64(lz.Responses.Responses3xx),
		fourHundredStatusRange:  int64(lz.Responses.Responses4xx),
		fiveHundredStatusRange:  int64(lz.Responses.Responses5xx),
	}

	nps.previousLocationZoneResponses[lzName] = respStatus
}

func (nps *NginxPlusScraper) recordHTTPLimitMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
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

func (nps *NginxPlusScraper) recordCacheMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for name, cache := range stats.Caches {
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Bypass.Bytes),
			metadata.AttributeNginxCacheOutcomeBYPASS,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Expired.Bytes),
			metadata.AttributeNginxCacheOutcomeEXPIRED,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Hit.Bytes),
			metadata.AttributeNginxCacheOutcomeHIT,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Miss.Bytes),
			metadata.AttributeNginxCacheOutcomeMISS,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Revalidated.Bytes),
			metadata.AttributeNginxCacheOutcomeREVALIDATED,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
			now,
			int64(cache.Stale.Bytes),
			metadata.AttributeNginxCacheOutcomeSTALE,
			name,
		)
		nps.mb.RecordNginxCacheBytesReadDataPoint(
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

func socketClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

// nolint: revive
func boolToInt64(booleanValue bool) int64 {
	if booleanValue {
		return 1
	}

	return 0
}
