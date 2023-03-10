/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	log "github.com/sirupsen/logrus"
)

const (
	// Peer state is one of “up”, “draining”, “down”, “unavail”, “checking”, and “unhealthy”.
	peerStateUp        = "up"
	peerStateDraining  = "draining"
	peerStateDown      = "down"
	peerStateUnavail   = "unavail"
	peerStateChecking  = "checking"
	peerStateUnhealthy = "unhealthy"
	valueFloat64One    = float64(1)
	valueFloat64Zero   = float64(0)
)

// NginxPlus generates metrics from NGINX Plus API
type NginxPlus struct {
	baseDimensions *metrics.CommonDim
	nginxNamespace,
	plusNamespace,
	plusAPI string
	// This is for keeping the previous stats.  Need to report the delta.
	prevStats     *plusclient.Stats
	init          sync.Once
	clientVersion int
	logger        *MetricSourceLogger
}

func NewNginxPlus(baseDimensions *metrics.CommonDim, nginxNamespace, plusNamespace, plusAPI string, clientVersion int) *NginxPlus {
	return &NginxPlus{baseDimensions: baseDimensions, nginxNamespace: nginxNamespace, plusNamespace: plusNamespace, plusAPI: plusAPI, clientVersion: clientVersion, logger: NewMetricSourceLogger()}
}

func (c *NginxPlus) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	c.init.Do(func() {
		cl, err := plusclient.NewNginxClientWithVersion(&http.Client{}, c.plusAPI, c.clientVersion)
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to create plus metrics client, %v", err))
			SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
			return
		}
		c.prevStats, err = cl.GetStats()
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to retrieve plus metrics, %v", err))
			SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
			c.prevStats = nil
			return
		}
	})

	cl, err := plusclient.NewNginxClientWithVersion(&http.Client{}, c.plusAPI, c.clientVersion)
	if err != nil {
		c.logger.Log(fmt.Sprintf("Failed to create plus metrics client, %v", err))
		SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
		return
	}

	stats, err := cl.GetStats()
	if err != nil {
		c.logger.Log(fmt.Sprintf("Failed to retrieve plus metrics, %v", err))
		SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
		return
	}

	if c.prevStats == nil {
		c.prevStats = stats
	}

	// TODO: check if we need these in collect
	c.baseDimensions.PublishedAPI = c.plusAPI
	c.baseDimensions.NginxType = c.plusNamespace
	c.baseDimensions.NginxBuild = stats.NginxInfo.Build
	c.baseDimensions.NginxVersion = stats.NginxInfo.Version

	c.sendMetrics(ctx, m, c.collectMetrics(stats, c.prevStats)...)

	c.prevStats = stats
}

func (c *NginxPlus) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.baseDimensions = dimensions
	c.plusAPI = collectorConf.PlusAPI
}

func (c *NginxPlus) Stop() {
	log.Debugf("Stopping NginxPlus source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxPlus) sendMetrics(ctx context.Context, m chan<- *proto.StatsEntity, entries ...*proto.StatsEntity) {
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		case m <- entry:
		}
	}
}

func (c *NginxPlus) collectMetrics(stats, prevStats *plusclient.Stats) (entries []*proto.StatsEntity) {
	entries = append(entries,
		c.instanceMetrics(stats, prevStats),
		c.commonMetrics(stats, prevStats),
		c.sslMetrics(stats, prevStats))
	entries = append(entries, c.serverZoneMetrics(stats, prevStats)...)
	entries = append(entries, c.streamServerZoneMetrics(stats, prevStats)...)
	entries = append(entries, c.locationZoneMetrics(stats, prevStats)...)
	entries = append(entries, c.cacheMetrics(stats, prevStats)...)
	entries = append(entries, c.httpUpstreamMetrics(stats, prevStats)...)
	entries = append(entries, c.streamUpstreamMetrics(stats, prevStats)...)
	entries = append(entries, c.slabMetrics(stats, prevStats)...)
	entries = append(entries, c.httpLimitConnsMetrics(stats, prevStats)...)
	entries = append(entries, c.httpLimitRequestMetrics(stats, prevStats)...)
	return entries
}

func (c *NginxPlus) instanceMetrics(stats, prevStats *plusclient.Stats) *proto.StatsEntity {
	l := &namedMetric{namespace: c.nginxNamespace, group: ""}

	configGeneration := stats.NginxInfo.Generation - prevStats.NginxInfo.Generation
	if stats.NginxInfo.Generation < prevStats.NginxInfo.Generation {
		configGeneration = stats.NginxInfo.Generation
	}

	simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
		"status":            float64(1.0),
		"config.generation": float64(configGeneration),
	})

	dims := c.baseDimensions.ToDimensions()
	return metrics.NewStatsEntity(dims, simpleMetrics)
}

// commonMetrics uses the namespace of nginx because the metrics are common between oss and plus
func (c *NginxPlus) commonMetrics(stats, prevStats *plusclient.Stats) *proto.StatsEntity {
	l := &namedMetric{namespace: c.nginxNamespace, group: "http"}

	// For the case if nginx restarted (systemctl restart nginx), the stats counters were reset to zero, and
	// the current counters will be less than previous counters.  Note, cannot just compare if the subtracted
	// value is less then zero, because some counters are UINT64, it just wrap around the negative number
	// to become a very big positive number.
	connAccepted := stats.Connections.Accepted - prevStats.Connections.Accepted
	if stats.Connections.Accepted < prevStats.Connections.Accepted {
		connAccepted = stats.Connections.Accepted
	}
	connDropped := stats.Connections.Dropped - prevStats.Connections.Dropped
	if stats.Connections.Dropped < prevStats.Connections.Dropped {
		connDropped = stats.Connections.Dropped
	}
	requestCount := stats.HTTPRequests.Total - prevStats.HTTPRequests.Total
	if stats.HTTPRequests.Total < prevStats.HTTPRequests.Total {
		requestCount = stats.HTTPRequests.Total
	}

	simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
		"conn.accepted":   float64(connAccepted),
		"conn.active":     float64(stats.Connections.Active),
		"conn.current":    float64(stats.Connections.Active + stats.Connections.Idle),
		"conn.dropped":    float64(connDropped),
		"conn.idle":       float64(stats.Connections.Idle),
		"request.current": float64(stats.HTTPRequests.Current),
		"request.count":   float64(requestCount),
	})

	dims := c.baseDimensions.ToDimensions()
	return metrics.NewStatsEntity(dims, simpleMetrics)
}

func (c *NginxPlus) sslMetrics(stats, prevStats *plusclient.Stats) *proto.StatsEntity {
	l := &namedMetric{namespace: c.plusNamespace, group: ""}

	sslHandshakes := stats.SSL.Handshakes - prevStats.SSL.Handshakes
	if stats.SSL.Handshakes < prevStats.SSL.Handshakes {
		sslHandshakes = stats.SSL.Handshakes
	}
	sslFailed := stats.SSL.HandshakesFailed - prevStats.SSL.HandshakesFailed
	if stats.SSL.HandshakesFailed < prevStats.SSL.HandshakesFailed {
		sslFailed = stats.SSL.HandshakesFailed
	}
	sslReuses := stats.SSL.SessionReuses - prevStats.SSL.SessionReuses
	if stats.SSL.SessionReuses < prevStats.SSL.SessionReuses {
		sslReuses = stats.SSL.SessionReuses
	}

	simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
		"ssl.handshakes": float64(sslHandshakes),
		"ssl.failed":     float64(sslFailed),
		"ssl.reuses":     float64(sslReuses),
	})

	dims := c.baseDimensions.ToDimensions()
	return metrics.NewStatsEntity(dims, simpleMetrics)
}

func (c *NginxPlus) serverZoneMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	zoneMetrics := make([]*proto.StatsEntity, 0)
	for name, sz := range stats.ServerZones {
		l := &namedMetric{namespace: c.plusNamespace, group: "http"}

		requestCount := sz.Requests - prevStats.ServerZones[name].Requests
		if sz.Requests < prevStats.ServerZones[name].Requests {
			requestCount = sz.Requests
		}
		responseCount := sz.Responses.Total - prevStats.ServerZones[name].Responses.Total
		if sz.Responses.Total < prevStats.ServerZones[name].Responses.Total {
			responseCount = sz.Responses.Total
		}
		statusDiscarded := sz.Discarded - prevStats.ServerZones[name].Discarded
		if sz.Discarded < prevStats.ServerZones[name].Discarded {
			statusDiscarded = sz.Discarded
		}
		requestBytesRcvd := sz.Received - prevStats.ServerZones[name].Received
		if sz.Received < prevStats.ServerZones[name].Received {
			requestBytesRcvd = sz.Received
		}
		requestBytesSent := sz.Sent - prevStats.ServerZones[name].Sent
		if sz.Sent < prevStats.ServerZones[name].Sent {
			requestBytesSent = sz.Sent
		}
		status1xx := sz.Responses.Responses1xx - prevStats.ServerZones[name].Responses.Responses1xx
		if sz.Responses.Responses1xx < prevStats.ServerZones[name].Responses.Responses1xx {
			status1xx = sz.Responses.Responses1xx
		}
		status2xx := sz.Responses.Responses2xx - prevStats.ServerZones[name].Responses.Responses2xx
		if sz.Responses.Responses2xx < prevStats.ServerZones[name].Responses.Responses2xx {
			status2xx = sz.Responses.Responses2xx
		}
		status3xx := sz.Responses.Responses3xx - prevStats.ServerZones[name].Responses.Responses3xx
		if sz.Responses.Responses3xx < prevStats.ServerZones[name].Responses.Responses3xx {
			status3xx = sz.Responses.Responses3xx
		}
		status4xx := sz.Responses.Responses4xx - prevStats.ServerZones[name].Responses.Responses4xx
		if sz.Responses.Responses4xx < prevStats.ServerZones[name].Responses.Responses4xx {
			status4xx = sz.Responses.Responses4xx
		}
		status5xx := sz.Responses.Responses5xx - prevStats.ServerZones[name].Responses.Responses5xx
		if sz.Responses.Responses5xx < prevStats.ServerZones[name].Responses.Responses5xx {
			status5xx = sz.Responses.Responses5xx
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"request.count":      float64(requestCount),
			"response.count":     float64(responseCount),
			"status.discarded":   float64(statusDiscarded),
			"status.processing":  float64(sz.Processing),
			"request.bytes_rcvd": float64(requestBytesRcvd),
			"request.bytes_sent": float64(requestBytesSent),
			"status.1xx":         float64(status1xx),
			"status.2xx":         float64(status2xx),
			"status.3xx":         float64(status3xx),
			"status.4xx":         float64(status4xx),
			"status.5xx":         float64(status5xx),
		})

		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "server_zone", Value: name})
		zoneMetrics = append(zoneMetrics, metrics.NewStatsEntity(dims, simpleMetrics))
	}

	return zoneMetrics
}

func (c *NginxPlus) streamServerZoneMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	zoneMetrics := make([]*proto.StatsEntity, 0)
	for name, ssz := range stats.StreamServerZones {
		l := &namedMetric{namespace: c.plusNamespace, group: "stream"}

		connections := ssz.Connections - prevStats.StreamServerZones[name].Connections
		if ssz.Connections < prevStats.StreamServerZones[name].Connections {
			connections = ssz.Connections
		}
		discarded := ssz.Discarded - prevStats.StreamServerZones[name].Discarded
		if ssz.Discarded < prevStats.StreamServerZones[name].Discarded {
			discarded = ssz.Discarded
		}
		bytesRcvd := ssz.Received - prevStats.StreamServerZones[name].Received
		if ssz.Received < prevStats.StreamServerZones[name].Received {
			bytesRcvd = ssz.Received
		}
		bytesSent := ssz.Sent - prevStats.StreamServerZones[name].Sent
		if ssz.Sent < prevStats.StreamServerZones[name].Sent {
			bytesSent = ssz.Sent
		}
		status2xx := ssz.Sessions.Sessions2xx - prevStats.StreamServerZones[name].Sessions.Sessions2xx
		if ssz.Sessions.Sessions2xx < prevStats.StreamServerZones[name].Sessions.Sessions2xx {
			status2xx = ssz.Sessions.Sessions2xx
		}
		status4xx := ssz.Sessions.Sessions4xx - prevStats.StreamServerZones[name].Sessions.Sessions4xx
		if ssz.Sessions.Sessions4xx < prevStats.StreamServerZones[name].Sessions.Sessions4xx {
			status4xx = ssz.Sessions.Sessions4xx
		}
		status5xx := ssz.Sessions.Sessions5xx - prevStats.StreamServerZones[name].Sessions.Sessions5xx
		if ssz.Sessions.Sessions5xx < prevStats.StreamServerZones[name].Sessions.Sessions5xx {
			status5xx = ssz.Sessions.Sessions5xx
		}
		statusTotal := ssz.Sessions.Total - prevStats.StreamServerZones[name].Sessions.Total
		if ssz.Sessions.Total < prevStats.StreamServerZones[name].Sessions.Total {
			statusTotal = ssz.Sessions.Total
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"connections":  float64(connections),
			"discarded":    float64(discarded),
			"processing":   float64(ssz.Processing),
			"bytes_rcvd":   float64(bytesRcvd),
			"bytes_sent":   float64(bytesSent),
			"status.2xx":   float64(status2xx),
			"status.4xx":   float64(status4xx),
			"status.5xx":   float64(status5xx),
			"status.total": float64(statusTotal),
		})

		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "server_zone", Value: name})
		zoneMetrics = append(zoneMetrics, metrics.NewStatsEntity(dims, simpleMetrics))
	}
	return zoneMetrics
}

func (c *NginxPlus) locationZoneMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	zoneMetrics := make([]*proto.StatsEntity, 0)
	for name, lz := range stats.LocationZones {
		l := &namedMetric{namespace: c.plusNamespace, group: "http"}

		statusDiscarded := lz.Discarded - prevStats.LocationZones[name].Discarded
		if lz.Discarded < prevStats.LocationZones[name].Discarded {
			statusDiscarded = lz.Discarded
		}
		requestCount := lz.Requests - prevStats.LocationZones[name].Requests
		if lz.Requests < prevStats.LocationZones[name].Requests {
			requestCount = lz.Requests
		}
		responseCount := lz.Responses.Total - prevStats.LocationZones[name].Responses.Total
		if lz.Responses.Total < prevStats.LocationZones[name].Responses.Total {
			responseCount = lz.Responses.Total
		}
		requestBytesRcvd := lz.Received - prevStats.LocationZones[name].Received
		if lz.Received < prevStats.LocationZones[name].Received {
			requestBytesRcvd = lz.Received
		}
		requestBytesSent := lz.Sent - prevStats.LocationZones[name].Sent
		if lz.Sent < prevStats.LocationZones[name].Sent {
			requestBytesSent = lz.Sent
		}
		status1xx := lz.Responses.Responses1xx - prevStats.LocationZones[name].Responses.Responses1xx
		if lz.Responses.Responses1xx < prevStats.LocationZones[name].Responses.Responses1xx {
			status1xx = lz.Responses.Responses1xx
		}
		status2xx := lz.Responses.Responses2xx - prevStats.LocationZones[name].Responses.Responses2xx
		if lz.Responses.Responses2xx < prevStats.LocationZones[name].Responses.Responses2xx {
			status2xx = lz.Responses.Responses2xx
		}
		status3xx := lz.Responses.Responses3xx - prevStats.LocationZones[name].Responses.Responses3xx
		if lz.Responses.Responses3xx < prevStats.LocationZones[name].Responses.Responses3xx {
			status3xx = lz.Responses.Responses3xx
		}
		status4xx := lz.Responses.Responses4xx - prevStats.LocationZones[name].Responses.Responses4xx
		if lz.Responses.Responses4xx < prevStats.LocationZones[name].Responses.Responses4xx {
			status4xx = lz.Responses.Responses4xx
		}
		status5xx := lz.Responses.Responses5xx - prevStats.LocationZones[name].Responses.Responses5xx
		if lz.Responses.Responses5xx < prevStats.LocationZones[name].Responses.Responses5xx {
			status5xx = lz.Responses.Responses5xx
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"status.discarded":   float64(statusDiscarded),
			"request.count":      float64(requestCount),
			"response.count":     float64(responseCount),
			"request.bytes_rcvd": float64(requestBytesRcvd),
			"request.bytes_sent": float64(requestBytesSent),
			"status.1xx":         float64(status1xx),
			"status.2xx":         float64(status2xx),
			"status.3xx":         float64(status3xx),
			"status.4xx":         float64(status4xx),
			"status.5xx":         float64(status5xx),
		})

		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "location_zone", Value: name})
		zoneMetrics = append(zoneMetrics, metrics.NewStatsEntity(dims, simpleMetrics))
	}

	return zoneMetrics
}

func (c *NginxPlus) httpUpstreamMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	upstreamMetrics := make([]*proto.StatsEntity, 0)
	for name, u := range stats.Upstreams {
		httpUpstreamHeaderTimes := []float64{}
		httpUpstreamResponseTimes := []float64{}
		l := &namedMetric{namespace: c.plusNamespace, group: "http"}
		peerStateMap := make(map[string]int)
		prevPeersMap := createHttpPeerMap(prevStats.Upstreams[name].Peers)
		for _, peer := range u.Peers {
			peerStateMap[peer.State] = peerStateMap[peer.State] + 1
			tempPeer := plusclient.Peer(peer)
			if prevPeer, ok := prevPeersMap[getHttpUpstreamPeerKey((peer))]; ok {
				if peer.Active >= prevPeer.Active {
					tempPeer.Active = peer.Active - prevPeer.Active
				}
				if peer.Requests >= prevPeer.Requests {
					tempPeer.Requests = peer.Requests - prevPeer.Requests
				}
				if peer.Responses.Total >= prevPeer.Responses.Total {
					tempPeer.Responses.Total = peer.Responses.Total - prevPeer.Responses.Total
				}
				if peer.Responses.Responses1xx >= prevPeer.Responses.Responses1xx {
					tempPeer.Responses.Responses1xx = peer.Responses.Responses1xx - prevPeer.Responses.Responses1xx
				}
				if peer.Responses.Responses2xx >= prevPeer.Responses.Responses2xx {
					tempPeer.Responses.Responses2xx = peer.Responses.Responses2xx - prevPeer.Responses.Responses2xx
				}
				if peer.Responses.Responses3xx >= prevPeer.Responses.Responses3xx {
					tempPeer.Responses.Responses3xx = peer.Responses.Responses3xx - prevPeer.Responses.Responses3xx
				}
				if peer.Responses.Responses4xx >= prevPeer.Responses.Responses4xx {
					tempPeer.Responses.Responses4xx = peer.Responses.Responses4xx - prevPeer.Responses.Responses4xx
				}
				if peer.Responses.Responses5xx >= prevPeer.Responses.Responses5xx {
					tempPeer.Responses.Responses5xx = peer.Responses.Responses5xx - prevPeer.Responses.Responses5xx
				}
				if peer.Sent >= prevPeer.Sent {
					tempPeer.Sent = peer.Sent - prevPeer.Sent
				}
				if peer.Received >= prevPeer.Received {
					tempPeer.Received = peer.Received - prevPeer.Received
				}
				if peer.Fails >= prevPeer.Fails {
					tempPeer.Fails = peer.Fails - prevPeer.Fails
				}
				if peer.Unavail >= prevPeer.Unavail {
					tempPeer.Unavail = peer.Unavail - prevPeer.Unavail
				}
				if peer.HealthChecks.Fails >= prevPeer.HealthChecks.Fails {
					tempPeer.HealthChecks.Fails = peer.HealthChecks.Fails - prevPeer.HealthChecks.Fails
				}
				if peer.HealthChecks.Unhealthy >= prevPeer.HealthChecks.Unhealthy {
					tempPeer.HealthChecks.Unhealthy = peer.HealthChecks.Unhealthy - prevPeer.HealthChecks.Unhealthy
				}
				if peer.HealthChecks.Checks >= prevPeer.HealthChecks.Checks {
					tempPeer.HealthChecks.Checks = peer.HealthChecks.Checks - prevPeer.HealthChecks.Checks
				}
			}

			httpUpstreamResponseTimes = append(httpUpstreamResponseTimes, float64(peer.ResponseTime))
			httpUpstreamHeaderTimes = append(httpUpstreamHeaderTimes, float64(peer.HeaderTime))

			simpleMetrics2 := l.convertSamplesToSimpleMetrics(map[string]float64{
				"upstream.peers.conn.active":             float64(tempPeer.Active),
				"upstream.peers.header_time":             float64(tempPeer.HeaderTime),
				"upstream.peers.response.time":           float64(tempPeer.ResponseTime),
				"upstream.peers.request.count":           float64(tempPeer.Requests),
				"upstream.peers.response.count":          float64(tempPeer.Responses.Total),
				"upstream.peers.status.1xx":              float64(tempPeer.Responses.Responses1xx),
				"upstream.peers.status.2xx":              float64(tempPeer.Responses.Responses2xx),
				"upstream.peers.status.3xx":              float64(tempPeer.Responses.Responses3xx),
				"upstream.peers.status.4xx":              float64(tempPeer.Responses.Responses4xx),
				"upstream.peers.status.5xx":              float64(tempPeer.Responses.Responses5xx),
				"upstream.peers.bytes_sent":              float64(tempPeer.Sent),
				"upstream.peers.bytes_rcvd":              float64(tempPeer.Received),
				"upstream.peers.fails":                   float64(tempPeer.Fails),
				"upstream.peers.unavail":                 float64(tempPeer.Unavail),
				"upstream.peers.health_checks.fails":     float64(tempPeer.HealthChecks.Fails),
				"upstream.peers.health_checks.unhealthy": float64(tempPeer.HealthChecks.Unhealthy),
				"upstream.peers.health_checks.checks":    float64(tempPeer.HealthChecks.Checks),
				"upstream.peers.state.up":                boolToFloat64(tempPeer.State == peerStateUp),
				"upstream.peers.state.draining":          boolToFloat64(tempPeer.State == peerStateDraining),
				"upstream.peers.state.down":              boolToFloat64(tempPeer.State == peerStateDown),
				"upstream.peers.state.unavail":           boolToFloat64(tempPeer.State == peerStateUnavail),
				"upstream.peers.state.checking":          boolToFloat64(tempPeer.State == peerStateChecking),
				"upstream.peers.state.unhealthy":         boolToFloat64(tempPeer.State == peerStateUnhealthy),
			})

			peerDims := c.baseDimensions.ToDimensions()
			peerDims = append(peerDims, &proto.Dimension{Name: "upstream", Value: name})
			peerDims = append(peerDims, &proto.Dimension{Name: "upstream_zone", Value: u.Zone})
			peerDims = append(peerDims, &proto.Dimension{Name: "peer.name", Value: peer.Name})
			peerDims = append(peerDims, &proto.Dimension{Name: "peer.address", Value: peer.Server})
			upstreamMetrics = append(upstreamMetrics, metrics.NewStatsEntity(peerDims, simpleMetrics2))
		}

		upstreamQueueOverflows := u.Queue.Overflows - prevStats.Upstreams[name].Queue.Overflows
		if u.Queue.Overflows < prevStats.Upstreams[name].Queue.Overflows {
			upstreamQueueOverflows = u.Queue.Overflows
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"upstream.keepalives":                 float64(u.Keepalives),
			"upstream.zombies":                    float64(u.Zombies),
			"upstream.queue.maxsize":              float64(u.Queue.MaxSize),
			"upstream.queue.overflows":            float64(upstreamQueueOverflows),
			"upstream.queue.size":                 float64(u.Queue.Size),
			"upstream.peers.total.up":             float64(peerStateMap[peerStateUp]),
			"upstream.peers.total.draining":       float64(peerStateMap[peerStateDraining]),
			"upstream.peers.total.down":           float64(peerStateMap[peerStateDown]),
			"upstream.peers.total.unavail":        float64(peerStateMap[peerStateUnavail]),
			"upstream.peers.total.checking":       float64(peerStateMap[peerStateChecking]),
			"upstream.peers.total.unhealthy":      float64(peerStateMap[peerStateUnhealthy]),
			"upstream.peers.response.time.count":  metrics.GetTimeMetrics(httpUpstreamResponseTimes, "count"),
			"upstream.peers.response.time.max":    metrics.GetTimeMetrics(httpUpstreamResponseTimes, "max"),
			"upstream.peers.response.time.median": metrics.GetTimeMetrics(httpUpstreamResponseTimes, "median"),
			"upstream.peers.response.time.pctl95": metrics.GetTimeMetrics(httpUpstreamResponseTimes, "pctl95"),
			"upstream.peers.header_time.count":    metrics.GetTimeMetrics(httpUpstreamHeaderTimes, "count"),
			"upstream.peers.header_time.max":      metrics.GetTimeMetrics(httpUpstreamHeaderTimes, "max"),
			"upstream.peers.header_time.median":   metrics.GetTimeMetrics(httpUpstreamHeaderTimes, "median"),
			"upstream.peers.header_time.pctl95":   metrics.GetTimeMetrics(httpUpstreamHeaderTimes, "pctl95"),
		})

		upstreamDims := c.baseDimensions.ToDimensions()
		upstreamDims = append(upstreamDims, &proto.Dimension{Name: "upstream", Value: name})
		upstreamDims = append(upstreamDims, &proto.Dimension{Name: "upstream_zone", Value: u.Zone})
		upstreamMetrics = append(upstreamMetrics, metrics.NewStatsEntity(upstreamDims, simpleMetrics))
	}

	return upstreamMetrics
}

func (c *NginxPlus) streamUpstreamMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	upstreamMetrics := make([]*proto.StatsEntity, 0)
	for name, u := range stats.StreamUpstreams {
		streamUpstreamResponseTimes := []float64{}
		streamUpstreamConnTimes := []float64{}
		l := &namedMetric{namespace: c.plusNamespace, group: "stream"}
		peerStateMap := make(map[string]int)
		prevPeersMap := createStreamPeerMap(prevStats.StreamUpstreams[name].Peers)
		for _, peer := range u.Peers {
			peerStateMap[peer.State] = peerStateMap[peer.State] + 1
			tempPeer := plusclient.StreamPeer(peer)
			if prevPeer, ok := prevPeersMap[getStreamUpstreamPeerKey((peer))]; ok {
				if peer.Active >= prevPeer.Active {
					tempPeer.Active = peer.Active - prevPeer.Active
				}
				if peer.Connections >= prevPeer.Connections {
					tempPeer.Connections = peer.Connections - prevPeer.Connections
				}
				if peer.Sent >= prevPeer.Sent {
					tempPeer.Sent = peer.Sent - prevPeer.Sent
				}
				if peer.Received >= prevPeer.Received {
					tempPeer.Received = peer.Received - prevPeer.Received
				}
				if peer.Fails >= prevPeer.Fails {
					tempPeer.Fails = peer.Fails - prevPeer.Fails
				}
				if peer.Unavail >= prevPeer.Unavail {
					tempPeer.Unavail = peer.Unavail - prevPeer.Unavail
				}
				if peer.HealthChecks.Fails >= prevPeer.HealthChecks.Fails {
					tempPeer.HealthChecks.Fails = peer.HealthChecks.Fails - prevPeer.HealthChecks.Fails
				}
				if peer.HealthChecks.Unhealthy >= prevPeer.HealthChecks.Unhealthy {
					tempPeer.HealthChecks.Unhealthy = peer.HealthChecks.Unhealthy - prevPeer.HealthChecks.Unhealthy
				}
				if peer.HealthChecks.Checks >= prevPeer.HealthChecks.Checks {
					tempPeer.HealthChecks.Checks = peer.HealthChecks.Checks - prevPeer.HealthChecks.Checks
				}
			}

			streamUpstreamResponseTimes = append(streamUpstreamResponseTimes, float64(peer.ResponseTime))
			streamUpstreamConnTimes = append(streamUpstreamConnTimes, float64(peer.ConnectTime))

			simpleMetrics2 := l.convertSamplesToSimpleMetrics(map[string]float64{
				"upstream.peers.conn.active":             float64(tempPeer.Active),
				"upstream.peers.conn.count":              float64(tempPeer.Connections),
				"upstream.peers.connect_time":            float64(tempPeer.ConnectTime),
				"upstream.peers.ttfb":                    float64(tempPeer.FirstByteTime),
				"upstream.peers.response.time":           float64(tempPeer.ResponseTime),
				"upstream.peers.bytes_sent":              float64(tempPeer.Sent),
				"upstream.peers.bytes_rcvd":              float64(tempPeer.Received),
				"upstream.peers.fails":                   float64(tempPeer.Fails),
				"upstream.peers.unavail":                 float64(tempPeer.Unavail),
				"upstream.peers.health_checks.fails":     float64(tempPeer.HealthChecks.Fails),
				"upstream.peers.health_checks.unhealthy": float64(tempPeer.HealthChecks.Unhealthy),
				"upstream.peers.health_checks.checks":    float64(tempPeer.HealthChecks.Checks),
				"upstream.peers.state.up":                boolToFloat64(tempPeer.State == peerStateUp),
				"upstream.peers.state.draining":          boolToFloat64(tempPeer.State == peerStateDraining),
				"upstream.peers.state.down":              boolToFloat64(tempPeer.State == peerStateDown),
				"upstream.peers.state.unavail":           boolToFloat64(tempPeer.State == peerStateUnavail),
				"upstream.peers.state.checking":          boolToFloat64(tempPeer.State == peerStateChecking),
				"upstream.peers.state.unhealthy":         boolToFloat64(tempPeer.State == peerStateUnhealthy),
			})

			peerDims := c.baseDimensions.ToDimensions()
			peerDims = append(peerDims, &proto.Dimension{Name: "upstream", Value: name})
			peerDims = append(peerDims, &proto.Dimension{Name: "upstream_zone", Value: u.Zone})
			peerDims = append(peerDims, &proto.Dimension{Name: "peer.name", Value: peer.Name})
			peerDims = append(peerDims, &proto.Dimension{Name: "peer.address", Value: peer.Server})
			upstreamMetrics = append(upstreamMetrics, metrics.NewStatsEntity(peerDims, simpleMetrics2))
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"upstream.zombies":                    float64(u.Zombies),
			"upstream.peers.total.up":             float64(peerStateMap[peerStateUp]),
			"upstream.peers.total.draining":       float64(peerStateMap[peerStateDraining]),
			"upstream.peers.total.down":           float64(peerStateMap[peerStateDown]),
			"upstream.peers.total.unavail":        float64(peerStateMap[peerStateUnavail]),
			"upstream.peers.total.checking":       float64(peerStateMap[peerStateChecking]),
			"upstream.peers.total.unhealthy":      float64(peerStateMap[peerStateUnhealthy]),
			"upstream.peers.response.time.count":  metrics.GetTimeMetrics(streamUpstreamResponseTimes, "count"),
			"upstream.peers.response.time.max":    metrics.GetTimeMetrics(streamUpstreamResponseTimes, "max"),
			"upstream.peers.response.time.median": metrics.GetTimeMetrics(streamUpstreamResponseTimes, "median"),
			"upstream.peers.response.time.pctl95": metrics.GetTimeMetrics(streamUpstreamResponseTimes, "pctl95"),
			"upstream.peers.connect_time.count":   metrics.GetTimeMetrics(streamUpstreamConnTimes, "count"),
			"upstream.peers.connect_time.max":     metrics.GetTimeMetrics(streamUpstreamConnTimes, "max"),
			"upstream.peers.connect_time.median":  metrics.GetTimeMetrics(streamUpstreamConnTimes, "median"),
			"upstream.peers.connect_time.pctl95":  metrics.GetTimeMetrics(streamUpstreamConnTimes, "pctl95"),
		})

		upstreamDims := c.baseDimensions.ToDimensions()
		upstreamDims = append(upstreamDims, &proto.Dimension{Name: "upstream", Value: name})
		upstreamDims = append(upstreamDims, &proto.Dimension{Name: "upstream_zone", Value: u.Zone})
		upstreamMetrics = append(upstreamMetrics, metrics.NewStatsEntity(upstreamDims, simpleMetrics))
	}

	return upstreamMetrics
}

func (c *NginxPlus) cacheMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	zoneMetrics := make([]*proto.StatsEntity, 0)
	for name, ca := range stats.Caches {
		l := &namedMetric{namespace: c.plusNamespace, group: "cache"}

		bypassResponses := ca.Bypass.Responses - prevStats.Caches[name].Bypass.Responses
		if ca.Bypass.Responses < prevStats.Caches[name].Bypass.Responses {
			bypassResponses = ca.Bypass.Responses
		}
		bypassBytes := ca.Bypass.Bytes - prevStats.Caches[name].Bypass.Bytes
		if ca.Bypass.Bytes < prevStats.Caches[name].Bypass.Bytes {
			bypassBytes = ca.Bypass.Bytes
		}
		expiredResponses := ca.Expired.Responses - prevStats.Caches[name].Expired.Responses
		if ca.Expired.Responses < prevStats.Caches[name].Expired.Responses {
			expiredResponses = ca.Expired.Responses
		}
		expiredBytes := ca.Expired.Bytes - prevStats.Caches[name].Expired.Bytes
		if ca.Expired.Bytes < prevStats.Caches[name].Expired.Bytes {
			expiredBytes = ca.Expired.Bytes
		}
		hitResponses := ca.Hit.Responses - prevStats.Caches[name].Hit.Responses
		if ca.Hit.Responses < prevStats.Caches[name].Hit.Responses {
			hitResponses = ca.Hit.Responses
		}
		hitBytes := ca.Hit.Bytes - prevStats.Caches[name].Hit.Bytes
		if ca.Hit.Bytes < prevStats.Caches[name].Hit.Bytes {
			hitBytes = ca.Hit.Bytes
		}
		missResponses := ca.Miss.Responses - prevStats.Caches[name].Miss.Responses
		if ca.Miss.Responses < prevStats.Caches[name].Miss.Responses {
			missResponses = ca.Miss.Responses
		}
		missBytes := ca.Miss.Bytes - prevStats.Caches[name].Miss.Bytes
		if ca.Miss.Bytes < prevStats.Caches[name].Miss.Bytes {
			missBytes = ca.Miss.Bytes
		}
		revalidatedResponses := ca.Revalidated.Responses - prevStats.Caches[name].Revalidated.Responses
		if ca.Revalidated.Responses < prevStats.Caches[name].Revalidated.Responses {
			revalidatedResponses = ca.Revalidated.Responses
		}
		revalidatedBytes := ca.Revalidated.Bytes - prevStats.Caches[name].Revalidated.Bytes
		if ca.Revalidated.Bytes < prevStats.Caches[name].Revalidated.Bytes {
			revalidatedBytes = ca.Revalidated.Bytes
		}
		staleResponses := ca.Stale.Responses - prevStats.Caches[name].Stale.Responses
		if ca.Stale.Responses < prevStats.Caches[name].Stale.Responses {
			staleResponses = ca.Stale.Responses
		}
		staleBytes := ca.Stale.Bytes - prevStats.Caches[name].Stale.Bytes
		if ca.Stale.Bytes < prevStats.Caches[name].Stale.Bytes {
			staleBytes = ca.Stale.Bytes
		}
		updatingResponses := ca.Updating.Responses - prevStats.Caches[name].Updating.Responses
		if ca.Updating.Responses < prevStats.Caches[name].Updating.Responses {
			updatingResponses = ca.Updating.Responses
		}
		updatingBytes := ca.Updating.Bytes - prevStats.Caches[name].Updating.Bytes
		if ca.Updating.Bytes < prevStats.Caches[name].Updating.Bytes {
			updatingBytes = ca.Updating.Bytes
		}

		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"size":                  float64(ca.Size),
			"max_size":              float64(ca.MaxSize),
			"bypass.responses":      float64(bypassResponses),
			"bypass.bytes":          float64(bypassBytes),
			"expired.responses":     float64(expiredResponses),
			"expired.bytes":         float64(expiredBytes),
			"hit.responses":         float64(hitResponses),
			"hit.bytes":             float64(hitBytes),
			"miss.responses":        float64(missResponses),
			"miss.bytes":            float64(missBytes),
			"revalidated.responses": float64(revalidatedResponses),
			"revalidated.bytes":     float64(revalidatedBytes),
			"stale.responses":       float64(staleResponses),
			"stale.bytes":           float64(staleBytes),
			"updating.responses":    float64(updatingResponses),
			"updating.bytes":        float64(updatingBytes),
		})

		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "cache_zone", Value: name})
		zoneMetrics = append(zoneMetrics, metrics.NewStatsEntity(dims, simpleMetrics))
	}

	return zoneMetrics
}

func (c *NginxPlus) slabMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	l := &namedMetric{namespace: c.plusNamespace, group: ""}
	slabMetrics := make([]*proto.StatsEntity, 0)

	for name, slab := range stats.Slabs {
		pages := slab.Pages
		used, free := pages.Used, pages.Free
		total := used + free
		var pctUsed float64
		if total > 0 {
			pctUsed = math.Round(float64(used) / float64(total) * 100)
		}

		slabSimpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"slab.pages.used":     float64(used),
			"slab.pages.free":     float64(free),
			"slab.pages.total":    float64(total),
			"slab.pages.pct_used": pctUsed,
		})

		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "zone", Value: name})
		slabMetrics = append(slabMetrics, metrics.NewStatsEntity(dims, slabSimpleMetrics))

		for slotNum, slot := range slab.Slots {
			slotSimpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
				"slab.slots." + slotNum + ".fails": float64(slot.Fails),
				"slab.slots." + slotNum + ".free":  float64(slot.Free),
				"slab.slots." + slotNum + ".reqs":  float64(slot.Reqs),
				"slab.slots." + slotNum + ".used":  float64(slot.Used),
			})
			slabMetrics = append(slabMetrics, metrics.NewStatsEntity(dims, slotSimpleMetrics))
		}
	}

	return slabMetrics
}

func (c *NginxPlus) httpLimitConnsMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	limitConnsMetrics := make([]*proto.StatsEntity, 0)

	for name, lc := range stats.HTTPLimitConnections {
		l := &namedMetric{namespace: c.plusNamespace, group: "http"}
		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"limit_conns.passed":           float64(lc.Passed - prevStats.HTTPLimitConnections[name].Passed),
			"limit_conns.rejected":         float64(lc.Rejected - prevStats.HTTPLimitConnections[name].Rejected),
			"limit_conns.rejected_dry_run": float64(lc.RejectedDryRun - prevStats.HTTPLimitConnections[name].RejectedDryRun),
		})
		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "limit_conn_zone", Value: name})
		limitConnsMetrics = append(limitConnsMetrics, metrics.NewStatsEntity(dims, simpleMetrics))

	}
	return limitConnsMetrics
}

func (c *NginxPlus) httpLimitRequestMetrics(stats, prevStats *plusclient.Stats) []*proto.StatsEntity {
	limitRequestMetrics := make([]*proto.StatsEntity, 0)

	for name, lr := range stats.HTTPLimitRequests {
		l := &namedMetric{namespace: c.plusNamespace, group: "http"}
		simpleMetrics := l.convertSamplesToSimpleMetrics(map[string]float64{
			"limit_reqs.passed":           float64(lr.Passed - prevStats.HTTPLimitRequests[name].Passed),
			"limit_reqs.delayed":          float64(lr.Delayed - prevStats.HTTPLimitRequests[name].Delayed),
			"limit_reqs.rejected":         float64(lr.Rejected - prevStats.HTTPLimitRequests[name].Rejected),
			"limit_reqs.delayed_dry_run":  float64(lr.DelayedDryRun - prevStats.HTTPLimitRequests[name].DelayedDryRun),
			"limit_reqs.rejected_dry_run": float64(lr.RejectedDryRun - prevStats.HTTPLimitRequests[name].RejectedDryRun),
		})
		dims := c.baseDimensions.ToDimensions()
		dims = append(dims, &proto.Dimension{Name: "limit_req_zone", Value: name})
		limitRequestMetrics = append(limitRequestMetrics, metrics.NewStatsEntity(dims, simpleMetrics))

	}
	return limitRequestMetrics
}

func getHttpUpstreamPeerKey(peer plusclient.Peer) (key string) {
	key = fmt.Sprintf("%s-%s-%s", peer.Server, peer.Service, peer.Name)
	return
}

func getStreamUpstreamPeerKey(peer plusclient.StreamPeer) (key string) {
	key = fmt.Sprintf("%s-%s-%s", peer.Server, peer.Service, peer.Name)
	return
}

func createHttpPeerMap(peers []plusclient.Peer) map[string]plusclient.Peer {
	m := make(map[string]plusclient.Peer, len(peers))
	for _, peer := range peers {
		m[getHttpUpstreamPeerKey(peer)] = peer
	}
	return m
}

func createStreamPeerMap(peers []plusclient.StreamPeer) map[string]plusclient.StreamPeer {
	m := make(map[string]plusclient.StreamPeer, len(peers))
	for _, peer := range peers {
		m[getStreamUpstreamPeerKey(peer)] = peer
	}
	return m
}

func boolToFloat64(mybool bool) float64 {
	if mybool {
		return valueFloat64One
	} else {
		return valueFloat64Zero
	}
}
