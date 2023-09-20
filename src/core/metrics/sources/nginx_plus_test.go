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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	currentHTTPRequestTotal          = 4567
	currentHTTPRequestCurrent        = 800
	previousHTTPRequestTotal         = 1234
	previousHTTPRequestCurrent       = 300
	currentZoneRequests              = 9
	currentZoneResponses             = 9
	currentZoneReceived              = 711
	currentZoneSent                  = 2106
	previousZoneRequests             = 6
	previousZoneResponses            = 6
	previousZoneReceived             = 443
	previousZoneSent                 = 1404
	cacheSize                        = 4096
	cacheMaxSize                     = 10737418240
	cacheHit                         = 4
	cacheMiss                        = 1
	cacheHitBytes                    = 5024
	cacheMissBytes                   = 1256
	currentSSLHandshakes             = 5
	currentSSLHandshakesFailed       = 5
	currentSSLSessionReuses          = 5
	previousSSLHandshakes            = 5
	previousSSLHandshakesFailed      = 5
	previousSSLSessionReuses         = 5
	upstreamQueueMaxSize             = 20
	currentPeer1UpstreamHeaderTime   = 100
	currentPeer2UpstreamHeaderTime   = 80
	currentPeer1UpstreamResponseTime = 100
	currentPeer2UpstreamResponseTime = 80
	currentUpstreamResponseTime      = 100
	currentUpstreamConnectTime       = 80
	currentUpstreamFirstByteTime     = 50
	previousUpstreamHeaderTime       = 98
	previousUpstreamResponseTime     = 98
	serverZoneName                   = "myserverzone1"
	streamServerZoneName             = "mystreamserverzone1"
	locationZoneName                 = "mylocationzone1"
	cacheZoneName                    = "mycachezone1"
	upstreamName                     = "myupstream"
	upstreamZoneName                 = "myupstreamzone1"
	limitRequestName                 = "mylimitreqszone"
	limitConnectionsName             = "mylimitconnszone"
	upstreamPeer1Name                = "127.0.0.1:9091"
	upstreamPeer1ServerAddress       = "127.0.0.1:9091"
	upstreamPeer2Name                = "f5.com"
	upstreamPeer2ServerAddress       = "127.0.0.1:9092"
	streamUpstreamPeer1Name          = "127.0.0.1:9093"
	streamUpstreamPeer1ServerAddress = "127.0.0.1:9093"
	streamUpstreamPeer2Name          = "127.0.0.1:9094"
	streamUpstreamPeer2ServerAddress = "127.0.0.1:9094"
	streamUpstreamPeer1ResponseTime  = 5
	streamUpstreamPeer2ResponseTime  = 2
	streamUpstreamPeer1ConnectTime   = 80
	streamUpstreamPeer2ConnectTime   = 100
	slabPageFree                     = 95
	slabPageUsed                     = 5
	slabPagePctUsed                  = 5
	currentWorkerConnAccepted        = 283
	currentWorkerConnDropped         = 21
	currentWorkerConnActive          = 8
	currentWorkerConnIdle            = 22
	currentWorkerHTTPRequestTotal    = 20022
	currentWorkerHTTPRequestCurrent  = 75
	previousWorkerConnAccepted       = 2
	previousWorkerConnDropped        = 5
	previousWorkerConnActive         = 8
	previousWorkerConnIdle           = 1
	previousWorkerHTTPRequestTotal   = 2001
	previousWorkerHTTPRequestCurrent = 21
	workerProcessID                  = 12345
)

type FakeNginxPlus struct {
	*NginxPlus
}

// Collect is fake collector that hard codes a stats struct response to avoid dependency on external NGINX Plus api
func (f *FakeNginxPlus) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()
	stats := plusclient.Stats{
		HTTPRequests: plusclient.HTTPRequests{
			Total:   currentHTTPRequestTotal,
			Current: currentHTTPRequestCurrent,
		},
		ServerZones: plusclient.ServerZones{
			serverZoneName: plusclient.ServerZone{
				Processing: 0,
				Requests:   currentZoneRequests,
				Responses: plusclient.Responses{
					Responses1xx: 0,
					Responses2xx: currentZoneResponses,
					Responses3xx: 0,
					Responses4xx: 0,
					Responses5xx: 0,
					Total:        currentZoneResponses,
				},
				Discarded: 0,
				Received:  currentZoneReceived,
				Sent:      currentZoneSent,
				SSL: plusclient.SSL{
					Handshakes:       currentSSLHandshakes,
					HandshakesFailed: currentSSLHandshakesFailed,
					SessionReuses:    currentSSLSessionReuses,
				},
			},
		},
		StreamServerZones: plusclient.StreamServerZones{
			streamServerZoneName: plusclient.StreamServerZone{
				Processing:  0,
				Connections: currentZoneRequests,
				Sessions: plusclient.Sessions{
					Sessions2xx: currentZoneResponses,
					Sessions4xx: 0,
					Sessions5xx: 0,
					Total:       currentZoneResponses,
				},
				Discarded: 0,
				Received:  currentZoneReceived,
				Sent:      currentZoneSent,
			},
		},
		LocationZones: plusclient.LocationZones{
			locationZoneName: plusclient.LocationZone{
				Requests: currentZoneRequests,
				Responses: plusclient.Responses{
					Responses1xx: 0,
					Responses2xx: currentZoneResponses,
					Responses3xx: 0,
					Responses4xx: 0,
					Responses5xx: 0,
					Total:        currentZoneResponses,
				},
				Discarded: 0,
				Received:  currentZoneReceived,
				Sent:      currentZoneSent,
			},
		},
		Caches: plusclient.Caches{
			cacheZoneName: plusclient.HTTPCache{
				Size:    cacheSize,
				MaxSize: cacheMaxSize,
				Cold:    false,
				Hit: plusclient.CacheStats{
					Responses: cacheHit,
					Bytes:     cacheHitBytes,
				},
				Stale: plusclient.CacheStats{
					Responses: 0,
					Bytes:     0,
				},
				Updating: plusclient.CacheStats{
					Responses: 0,
					Bytes:     0,
				},
				Revalidated: plusclient.CacheStats{
					Responses: 0,
					Bytes:     0,
				},
				Miss: plusclient.CacheStats{
					Responses: cacheMiss,
					Bytes:     cacheMissBytes,
				},
				Expired: plusclient.ExtendedCacheStats{
					CacheStats: plusclient.CacheStats{
						Responses: 0,
						Bytes:     0,
					},
					ResponsesWritten: 0,
					BytesWritten:     0,
				},
				Bypass: plusclient.ExtendedCacheStats{
					CacheStats: plusclient.CacheStats{
						Responses: 0,
						Bytes:     0,
					},
					ResponsesWritten: 0,
					BytesWritten:     0,
				},
			},
		},
		Upstreams: plusclient.Upstreams{
			upstreamName: plusclient.Upstream{
				Peers: []plusclient.Peer{
					{
						ID:     0,
						Server: upstreamPeer1ServerAddress,
						Name:   upstreamPeer1Name,
						Backup: false,
						Weight: 1,
						State:  "up",
						SSL: plusclient.SSL{
							Handshakes:       currentSSLHandshakes,
							HandshakesFailed: currentSSLHandshakesFailed,
							SessionReuses:    currentSSLSessionReuses,
						},
						Requests: currentZoneRequests,
						Responses: plusclient.Responses{
							Responses1xx: 0,
							Responses2xx: currentZoneResponses,
							Responses3xx: 0,
							Responses4xx: 0,
							Responses5xx: 0,
							Total:        currentZoneResponses,
						},
						Sent:     currentZoneSent,
						Received: currentZoneReceived,
						Fails:    0,
						Unavail:  0,
						HealthChecks: plusclient.HealthChecks{
							Checks:     0,
							Fails:      0,
							Unhealthy:  0,
							LastPassed: false,
						},
						HeaderTime:   currentPeer1UpstreamHeaderTime,
						ResponseTime: currentPeer1UpstreamResponseTime,
					},
					{
						ID:     1,
						Server: upstreamPeer2ServerAddress,
						Name:   upstreamPeer2Name,
						Backup: false,
						Weight: 1,
						State:  "up",
						SSL: plusclient.SSL{
							Handshakes:       currentSSLHandshakes,
							HandshakesFailed: currentSSLHandshakesFailed,
							SessionReuses:    currentSSLSessionReuses,
						},
						Requests: currentZoneRequests,
						Responses: plusclient.Responses{
							Responses1xx: 0,
							Responses2xx: currentZoneResponses,
							Responses3xx: 0,
							Responses4xx: 0,
							Responses5xx: 0,
							Total:        currentZoneResponses,
						},
						Sent:     currentZoneSent,
						Received: currentZoneReceived,
						Fails:    0,
						Unavail:  0,
						HealthChecks: plusclient.HealthChecks{
							Checks:     0,
							Fails:      0,
							Unhealthy:  0,
							LastPassed: false,
						},
						HeaderTime:   currentPeer2UpstreamHeaderTime,
						ResponseTime: currentPeer2UpstreamResponseTime,
					},
				},
				Keepalives: 0,
				Zombies:    0,
				Zone:       upstreamZoneName,
				Queue: plusclient.Queue{
					Size:      0,
					MaxSize:   upstreamQueueMaxSize,
					Overflows: 0,
				},
			},
		},
		StreamUpstreams: plusclient.StreamUpstreams{
			upstreamName: plusclient.StreamUpstream{
				Peers: []plusclient.StreamPeer{
					{
						ID:            0,
						Server:        streamUpstreamPeer1ServerAddress,
						Name:          streamUpstreamPeer1Name,
						Backup:        false,
						Weight:        1,
						State:         "up",
						Connections:   1,
						ConnectTime:   streamUpstreamPeer1ConnectTime,
						FirstByteTime: currentUpstreamFirstByteTime,
						ResponseTime:  streamUpstreamPeer1ResponseTime,
						Sent:          currentZoneSent,
						Received:      currentZoneReceived,
						Fails:         0,
						Unavail:       0,
						HealthChecks: plusclient.HealthChecks{
							Checks:    0,
							Fails:     0,
							Unhealthy: 0,
						},
					},
					{
						ID:            1,
						Server:        streamUpstreamPeer2ServerAddress,
						Name:          streamUpstreamPeer2Name,
						Backup:        false,
						Weight:        1,
						State:         "up",
						Connections:   1,
						ConnectTime:   streamUpstreamPeer2ConnectTime,
						FirstByteTime: currentUpstreamFirstByteTime,
						ResponseTime:  streamUpstreamPeer2ResponseTime,
						Sent:          currentZoneSent,
						Received:      currentZoneReceived,
						Fails:         0,
						Unavail:       0,
						HealthChecks: plusclient.HealthChecks{
							Checks:    0,
							Fails:     0,
							Unhealthy: 0,
						},
					},
				},
				Zombies: 0,
				Zone:    upstreamZoneName,
			},
		},
		Slabs: plusclient.Slabs{
			serverZoneName: plusclient.Slab{
				Pages: plusclient.Pages{
					Used: slabPageUsed,
					Free: slabPageFree,
				},
				Slots: plusclient.Slots{
					"8": plusclient.Slot{
						Used:  1,
						Free:  2,
						Reqs:  3,
						Fails: 4,
					},
				},
			},
		},
		HTTPLimitConnections: plusclient.HTTPLimitConnections{
			limitConnectionsName: plusclient.LimitConnection{
				Passed:         15,
				Rejected:       0,
				RejectedDryRun: 2,
			},
		},
		HTTPLimitRequests: plusclient.HTTPLimitRequests{
			limitRequestName: plusclient.HTTPLimitRequest{
				Passed:         15,
				Rejected:       0,
				Delayed:        4,
				DelayedDryRun:  4,
				RejectedDryRun: 2,
			},
		},
		Workers: []*plusclient.Workers{
			{
				ProcessID: 12345,
				HTTP: plusclient.WorkersHTTP{
					HTTPRequests: plusclient.HTTPRequests{
						Total:   currentWorkerHTTPRequestTotal,
						Current: currentWorkerHTTPRequestCurrent,
					},
				},
				Connections: plusclient.Connections{
					Accepted: currentWorkerConnAccepted,
					Dropped:  currentWorkerConnDropped,
					Active:   currentWorkerConnActive,
					Idle:     currentWorkerConnIdle,
				},
			},
		},
	}

	prevStats := plusclient.Stats{
		HTTPRequests: plusclient.HTTPRequests{
			Total:   previousHTTPRequestTotal,
			Current: previousHTTPRequestCurrent,
		},
		ServerZones: plusclient.ServerZones{
			serverZoneName: plusclient.ServerZone{
				Processing: 0,
				Requests:   previousZoneRequests,
				Responses: plusclient.Responses{
					Responses1xx: 0,
					Responses2xx: previousZoneResponses,
					Responses3xx: 0,
					Responses4xx: 0,
					Responses5xx: 0,
					Total:        previousZoneResponses,
				},
				Discarded: 0,
				Received:  previousZoneReceived,
				Sent:      previousZoneSent,
				SSL: plusclient.SSL{
					Handshakes:       previousSSLHandshakes,
					HandshakesFailed: previousSSLHandshakesFailed,
					SessionReuses:    previousSSLSessionReuses,
				},
			},
		},
		LocationZones: plusclient.LocationZones{
			locationZoneName: plusclient.LocationZone{
				Requests: previousZoneRequests + currentZoneRequests, // to test the systemctl restart case where the prevStats is bigger than currentStats
				Responses: plusclient.Responses{
					Responses1xx: 0,
					Responses2xx: previousZoneResponses + currentZoneResponses,
					Responses3xx: 0,
					Responses4xx: 0,
					Responses5xx: 0,
					Total:        previousZoneResponses + currentZoneResponses,
				},
				Discarded: 0,
				Received:  previousZoneReceived + currentZoneReceived,
				Sent:      previousZoneSent + currentZoneReceived,
			},
		},
		Upstreams: plusclient.Upstreams{
			upstreamName: plusclient.Upstream{
				Peers: []plusclient.Peer{
					{
						ID:     0,
						Server: upstreamPeer1ServerAddress,
						Name:   upstreamPeer1Name,
						Backup: false,
						Weight: 1,
						State:  "up",
						SSL: plusclient.SSL{
							Handshakes:       previousSSLHandshakes,
							HandshakesFailed: previousSSLHandshakesFailed,
							SessionReuses:    previousSSLSessionReuses,
						},
						Requests: previousZoneRequests,
						Responses: plusclient.Responses{
							Responses1xx: 0,
							Responses2xx: previousZoneResponses,
							Responses3xx: 0,
							Responses4xx: 0,
							Responses5xx: 0,
							Total:        previousZoneResponses,
						},
						Sent:     previousZoneSent,
						Received: previousZoneReceived,
						Fails:    0,
						Unavail:  0,
						HealthChecks: plusclient.HealthChecks{
							Checks:     0,
							Fails:      0,
							Unhealthy:  0,
							LastPassed: false,
						},
						HeaderTime:   previousUpstreamHeaderTime,
						ResponseTime: previousUpstreamResponseTime,
					},
					{
						ID:     1,
						Server: upstreamPeer2ServerAddress,
						Name:   upstreamPeer2Name,
						Backup: false,
						Weight: 1,
						State:  "up",
						SSL: plusclient.SSL{
							Handshakes:       previousSSLHandshakes,
							HandshakesFailed: previousSSLHandshakesFailed,
							SessionReuses:    previousSSLSessionReuses,
						},
						Requests: previousZoneRequests,
						Responses: plusclient.Responses{
							Responses1xx: 0,
							Responses2xx: previousZoneResponses,
							Responses3xx: 0,
							Responses4xx: 0,
							Responses5xx: 0,
							Total:        previousZoneResponses,
						},
						Sent:     previousZoneSent,
						Received: previousZoneReceived,
						Fails:    0,
						Unavail:  0,
						HealthChecks: plusclient.HealthChecks{
							Checks:     0,
							Fails:      0,
							Unhealthy:  0,
							LastPassed: false,
						},
						HeaderTime:   previousUpstreamHeaderTime,
						ResponseTime: previousUpstreamResponseTime,
					},
				},
				Keepalives: 0,
				Zombies:    0,
				Zone:       upstreamZoneName,
				Queue: plusclient.Queue{
					Size:      0,
					MaxSize:   upstreamQueueMaxSize,
					Overflows: 0,
				},
			},
		},
		HTTPLimitConnections: plusclient.HTTPLimitConnections{
			limitConnectionsName: plusclient.LimitConnection{
				Passed:         5,
				Rejected:       0,
				RejectedDryRun: 0,
			},
		},
		HTTPLimitRequests: plusclient.HTTPLimitRequests{
			limitRequestName: plusclient.HTTPLimitRequest{
				Passed:         5,
				Rejected:       0,
				Delayed:        2,
				DelayedDryRun:  3,
				RejectedDryRun: 0,
			},
		},
		Workers: []*plusclient.Workers{
			{
				ProcessID: 12345,
				HTTP: plusclient.WorkersHTTP{
					HTTPRequests: plusclient.HTTPRequests{
						Total:   previousWorkerHTTPRequestTotal,
						Current: previousWorkerHTTPRequestCurrent,
					},
				},
				Connections: plusclient.Connections{
					Accepted: previousWorkerConnAccepted,
					Dropped:  previousWorkerConnDropped,
					Active:   previousWorkerConnActive,
					Idle:     previousWorkerConnIdle,
				},
			},
		},
	}

	f.baseDimensions.NginxType = "plus"
	f.baseDimensions.PublishedAPI = f.plusAPI
	f.baseDimensions.NginxBuild = stats.NginxInfo.Build
	f.baseDimensions.NginxVersion = stats.NginxInfo.Version

	f.sendMetrics(ctx, m, f.collectMetrics(&stats, &prevStats)...)
}

func TestNginxPlusUpdate(t *testing.T) {
	nginxPlus := NewNginxPlus(&metrics.CommonDim{}, "test", PlusNamespace, "http://localhost:8080/api", 6)

	assert.Equal(t, "", nginxPlus.baseDimensions.InstanceTags)
	assert.Equal(t, "http://localhost:8080/api", nginxPlus.plusAPI)

	nginxPlus.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{
			PlusAPI: "http://localhost:8080/new_api",
		},
	)

	assert.Equal(t, "new-tag", nginxPlus.baseDimensions.InstanceTags)
	assert.Equal(t, "http://localhost:8080/new_api", nginxPlus.plusAPI)
}

func TestNginxPlus_Collect(t *testing.T) {
	expectedInstanceMetrics := map[string]float64{
		"nginx.status":            1,
		"nginx.config.generation": 0,
	}

	expectedCommonMetrics := map[string]float64{
		"nginx.http.conn.active":     0,
		"nginx.http.conn.accepted":   0,
		"nginx.http.conn.current":    0,
		"nginx.http.conn.dropped":    0,
		"nginx.http.conn.idle":       0,
		"nginx.http.request.current": currentHTTPRequestCurrent,
		"nginx.http.request.count":   currentHTTPRequestTotal - previousHTTPRequestTotal,
	}

	expectedSSLMetrics := map[string]float64{
		"plus.ssl.handshakes": 0,
		"plus.ssl.failed":     0,
		"plus.ssl.reuses":     0,
	}

	expectedServerZoneMetrics := map[string]float64{
		"plus.http.request.count":         currentZoneRequests - previousZoneRequests,
		"plus.http.response.count":        currentZoneResponses - previousZoneResponses,
		"plus.http.status.discarded":      0,
		"plus.http.status.processing":     0,
		"plus.http.request.bytes_rcvd":    currentZoneReceived - previousZoneReceived,
		"plus.http.request.bytes_sent":    currentZoneSent - previousZoneSent,
		"plus.http.status.1xx":            0,
		"plus.http.status.2xx":            currentZoneResponses - previousZoneResponses,
		"plus.http.status.3xx":            0,
		"plus.http.status.4xx":            0,
		"plus.http.status.5xx":            0,
		"plus.http.ssl.handshakes":        currentSSLHandshakes - previousSSLHandshakes,
		"plus.http.ssl.handshakes.failed": currentSSLHandshakesFailed - previousSSLHandshakesFailed,
		"plus.http.ssl.session.reuses":    currentSSLSessionReuses - previousSSLSessionReuses,
	}

	expectedStreamServerZoneMetrics := map[string]float64{
		"plus.stream.connections":  currentZoneRequests,
		"plus.stream.discarded":    0,
		"plus.stream.processing":   0,
		"plus.stream.bytes_rcvd":   currentZoneReceived,
		"plus.stream.bytes_sent":   currentZoneSent,
		"plus.stream.status.2xx":   currentZoneResponses,
		"plus.stream.status.4xx":   0,
		"plus.stream.status.5xx":   0,
		"plus.stream.status.total": currentZoneResponses,
	}

	expectedLocationZoneMetrics := map[string]float64{
		"plus.http.status.discarded":   0,
		"plus.http.request.count":      currentZoneRequests,
		"plus.http.response.count":     currentZoneResponses,
		"plus.http.request.bytes_rcvd": currentZoneReceived,
		"plus.http.request.bytes_sent": currentZoneSent,
		"plus.http.status.1xx":         0,
		"plus.http.status.2xx":         currentZoneResponses,
		"plus.http.status.3xx":         0,
		"plus.http.status.4xx":         0,
		"plus.http.status.5xx":         0,
	}

	expectedCacheZoneMetrics := map[string]float64{
		"plus.cache.size":                  cacheSize,
		"plus.cache.max_size":              cacheMaxSize,
		"plus.cache.bypass.responses":      0,
		"plus.cache.bypass.bytes":          0,
		"plus.cache.expired.responses":     0,
		"plus.cache.expired.bytes":         0,
		"plus.cache.hit.responses":         cacheHit,
		"plus.cache.hit.bytes":             cacheHitBytes,
		"plus.cache.miss.responses":        cacheMiss,
		"plus.cache.miss.bytes":            cacheMissBytes,
		"plus.cache.revalidated.responses": 0,
		"plus.cache.revalidated.bytes":     0,
		"plus.cache.stale.responses":       0,
		"plus.cache.stale.bytes":           0,
		"plus.cache.updating.responses":    0,
		"plus.cache.updating.bytes":        0,
	}

	expectedHttpUpstreamMetrics := map[string]float64{
		"plus.http.upstream.keepalives":                 0,
		"plus.http.upstream.zombies":                    0,
		"plus.http.upstream.queue.maxsize":              upstreamQueueMaxSize,
		"plus.http.upstream.queue.overflows":            0,
		"plus.http.upstream.queue.size":                 0,
		"plus.http.upstream.peers.total.up":             2,
		"plus.http.upstream.peers.total.draining":       0,
		"plus.http.upstream.peers.total.down":           0,
		"plus.http.upstream.peers.total.unavail":        0,
		"plus.http.upstream.peers.total.checking":       0,
		"plus.http.upstream.peers.total.unhealthy":      0,
		"plus.http.upstream.peers.header_time.count":    2,
		"plus.http.upstream.peers.header_time.max":      100,
		"plus.http.upstream.peers.header_time.median":   90,
		"plus.http.upstream.peers.header_time.pctl95":   100,
		"plus.http.upstream.peers.response.time.count":  2,
		"plus.http.upstream.peers.response.time.max":    100,
		"plus.http.upstream.peers.response.time.median": 90,
		"plus.http.upstream.peers.response.time.pctl95": 100,
	}

	expectedHttpPeer1UpstreamMetrics := map[string]float64{
		"plus.http.upstream.peers.conn.active":             0,
		"plus.http.upstream.peers.header_time":             currentPeer1UpstreamHeaderTime,
		"plus.http.upstream.peers.response.time":           currentPeer1UpstreamResponseTime,
		"plus.http.upstream.peers.request.count":           currentZoneRequests - previousZoneRequests,
		"plus.http.upstream.peers.response.count":          currentZoneResponses - previousZoneResponses,
		"plus.http.upstream.peers.status.1xx":              0,
		"plus.http.upstream.peers.status.2xx":              currentZoneResponses - previousZoneResponses,
		"plus.http.upstream.peers.status.3xx":              0,
		"plus.http.upstream.peers.status.4xx":              0,
		"plus.http.upstream.peers.status.5xx":              0,
		"plus.http.upstream.peers.bytes_sent":              currentZoneSent - previousZoneSent,
		"plus.http.upstream.peers.bytes_rcvd":              currentZoneReceived - previousZoneReceived,
		"plus.http.upstream.peers.fails":                   0,
		"plus.http.upstream.peers.unavail":                 0,
		"plus.http.upstream.peers.health_checks.fails":     0,
		"plus.http.upstream.peers.health_checks.unhealthy": 0,
		"plus.http.upstream.peers.health_checks.checks":    0,
		"plus.http.upstream.peers.state.up":                1,
		"plus.http.upstream.peers.state.draining":          0,
		"plus.http.upstream.peers.state.down":              0,
		"plus.http.upstream.peers.state.unavail":           0,
		"plus.http.upstream.peers.state.checking":          0,
		"plus.http.upstream.peers.state.unhealthy":         0,
		"plus.http.upstream.peers.ssl.handshakes":          currentSSLHandshakes - previousSSLHandshakes,
		"plus.http.upstream.peers.ssl.handshakes.failed":   currentSSLHandshakesFailed - previousSSLHandshakesFailed,
		"plus.http.upstream.peers.ssl.session.reuses":      currentSSLSessionReuses - previousSSLSessionReuses,
	}

	expectedHttpPeer2UpstreamMetrics := map[string]float64{
		"plus.http.upstream.peers.conn.active":             0,
		"plus.http.upstream.peers.header_time":             currentPeer2UpstreamHeaderTime,
		"plus.http.upstream.peers.response.time":           currentPeer2UpstreamResponseTime,
		"plus.http.upstream.peers.request.count":           currentZoneRequests - previousZoneRequests,
		"plus.http.upstream.peers.response.count":          currentZoneResponses - previousZoneResponses,
		"plus.http.upstream.peers.status.1xx":              0,
		"plus.http.upstream.peers.status.2xx":              currentZoneResponses - previousZoneResponses,
		"plus.http.upstream.peers.status.3xx":              0,
		"plus.http.upstream.peers.status.4xx":              0,
		"plus.http.upstream.peers.status.5xx":              0,
		"plus.http.upstream.peers.bytes_sent":              currentZoneSent - previousZoneSent,
		"plus.http.upstream.peers.bytes_rcvd":              currentZoneReceived - previousZoneReceived,
		"plus.http.upstream.peers.fails":                   0,
		"plus.http.upstream.peers.unavail":                 0,
		"plus.http.upstream.peers.health_checks.fails":     0,
		"plus.http.upstream.peers.health_checks.unhealthy": 0,
		"plus.http.upstream.peers.health_checks.checks":    0,
		"plus.http.upstream.peers.state.up":                1,
		"plus.http.upstream.peers.state.draining":          0,
		"plus.http.upstream.peers.state.down":              0,
		"plus.http.upstream.peers.state.unavail":           0,
		"plus.http.upstream.peers.state.checking":          0,
		"plus.http.upstream.peers.state.unhealthy":         0,
		"plus.http.upstream.peers.ssl.handshakes":          currentSSLHandshakes - previousSSLHandshakes,
		"plus.http.upstream.peers.ssl.handshakes.failed":   currentSSLHandshakesFailed - previousSSLHandshakesFailed,
		"plus.http.upstream.peers.ssl.session.reuses":      currentSSLSessionReuses - previousSSLSessionReuses,
	}

	expectedStreamUpstreamMetrics := map[string]float64{
		"plus.stream.upstream.zombies":                    0,
		"plus.stream.upstream.peers.total.up":             2,
		"plus.stream.upstream.peers.total.draining":       0,
		"plus.stream.upstream.peers.total.down":           0,
		"plus.stream.upstream.peers.total.unavail":        0,
		"plus.stream.upstream.peers.total.checking":       0,
		"plus.stream.upstream.peers.total.unhealthy":      0,
		"plus.stream.upstream.peers.response.time.count":  2,
		"plus.stream.upstream.peers.response.time.max":    5,
		"plus.stream.upstream.peers.response.time.median": 3.5,
		"plus.stream.upstream.peers.response.time.pctl95": 5,
		"plus.stream.upstream.peers.connect_time.count":   2,
		"plus.stream.upstream.peers.connect_time.max":     100,
		"plus.stream.upstream.peers.connect_time.median":  90,
		"plus.stream.upstream.peers.connect_time.pctl95":  100,
	}

	expectedStreamPeer1UpstreamMetrics := map[string]float64{
		"plus.stream.upstream.peers.conn.active":             0,
		"plus.stream.upstream.peers.conn.count":              1,
		"plus.stream.upstream.peers.connect_time":            streamUpstreamPeer1ConnectTime,
		"plus.stream.upstream.peers.ttfb":                    currentUpstreamFirstByteTime,
		"plus.stream.upstream.peers.response.time":           streamUpstreamPeer1ResponseTime,
		"plus.stream.upstream.peers.bytes_sent":              currentZoneSent,
		"plus.stream.upstream.peers.bytes_rcvd":              currentZoneReceived,
		"plus.stream.upstream.peers.fails":                   0,
		"plus.stream.upstream.peers.unavail":                 0,
		"plus.stream.upstream.peers.health_checks.fails":     0,
		"plus.stream.upstream.peers.health_checks.unhealthy": 0,
		"plus.stream.upstream.peers.health_checks.checks":    0,
		"plus.stream.upstream.peers.state.up":                1,
		"plus.stream.upstream.peers.state.draining":          0,
		"plus.stream.upstream.peers.state.down":              0,
		"plus.stream.upstream.peers.state.unavail":           0,
		"plus.stream.upstream.peers.state.checking":          0,
		"plus.stream.upstream.peers.state.unhealthy":         0,
	}

	expectedStreamPeer2UpstreamMetrics := map[string]float64{
		"plus.stream.upstream.peers.conn.active":             0,
		"plus.stream.upstream.peers.conn.count":              1,
		"plus.stream.upstream.peers.connect_time":            streamUpstreamPeer2ConnectTime,
		"plus.stream.upstream.peers.ttfb":                    currentUpstreamFirstByteTime,
		"plus.stream.upstream.peers.response.time":           streamUpstreamPeer2ResponseTime,
		"plus.stream.upstream.peers.bytes_sent":              currentZoneSent,
		"plus.stream.upstream.peers.bytes_rcvd":              currentZoneReceived,
		"plus.stream.upstream.peers.fails":                   0,
		"plus.stream.upstream.peers.unavail":                 0,
		"plus.stream.upstream.peers.health_checks.fails":     0,
		"plus.stream.upstream.peers.health_checks.unhealthy": 0,
		"plus.stream.upstream.peers.health_checks.checks":    0,
		"plus.stream.upstream.peers.state.up":                1,
		"plus.stream.upstream.peers.state.draining":          0,
		"plus.stream.upstream.peers.state.down":              0,
		"plus.stream.upstream.peers.state.unavail":           0,
		"plus.stream.upstream.peers.state.checking":          0,
		"plus.stream.upstream.peers.state.unhealthy":         0,
	}

	expectedSlabMetrics := map[string]float64{
		"plus.slab.pages.used":     slabPageUsed,
		"plus.slab.pages.free":     slabPageFree,
		"plus.slab.pages.total":    slabPageFree + slabPageUsed,
		"plus.slab.pages.pct_used": slabPagePctUsed,
	}

	expectedSlabSlotMetrics := map[string]float64{
		"plus.slab.slots.8.used":  1,
		"plus.slab.slots.8.free":  2,
		"plus.slab.slots.8.reqs":  3,
		"plus.slab.slots.8.fails": 4,
	}

	expectedHTTPLimitConnsMetrics := map[string]float64{
		"plus.http.limit_conns.passed":           10,
		"plus.http.limit_conns.rejected":         0,
		"plus.http.limit_conns.rejected_dry_run": 2,
	}

	expectedHTTPLimitReqsMetrics := map[string]float64{
		"plus.http.limit_reqs.passed":           10,
		"plus.http.limit_reqs.delayed":          2,
		"plus.http.limit_reqs.rejected":         0,
		"plus.http.limit_reqs.delayed_dry_run":  1,
		"plus.http.limit_reqs.rejected_dry_run": 2,
	}

	expectedWorkerMetrics := map[string]float64{
		"plus.worker.conn.accepted":        currentWorkerConnAccepted - previousWorkerConnAccepted,
		"plus.worker.conn.dropped":         currentWorkerConnDropped - previousWorkerConnDropped,
		"plus.worker.conn.active":          currentWorkerConnActive - previousWorkerConnActive,
		"plus.worker.conn.idle":            currentWorkerConnIdle - previousWorkerConnIdle,
		"plus.worker.http.request.total":   currentWorkerHTTPRequestTotal - previousWorkerHTTPRequestTotal,
		"plus.worker.http.request.current": currentWorkerHTTPRequestCurrent - previousWorkerHTTPRequestCurrent,
	}

	tests := []struct {
		baseDimensions *metrics.CommonDim
		m              chan *metrics.StatsEntityWrapper
	}{
		{
			baseDimensions: metrics.NewCommonDim(
				&proto.HostInfo{
					Hostname: "MyServer",
				},
				&config.Config{},
				"",
			),
			m: make(chan *metrics.StatsEntityWrapper, 127),
		},
	}

	for _, test := range tests {
		ctx := context.TODO()

		f := &FakeNginxPlus{NewNginxPlus(test.baseDimensions, "nginx", "plus", "", 6)}
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go f.Collect(ctx, wg, test.m)
		wg.Wait()

		instanceMetrics := <-test.m
		assert.Len(t, instanceMetrics.Data.Simplemetrics, len(expectedInstanceMetrics))
		for _, metric := range instanceMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedInstanceMetrics, metric.Name)
			assert.Equal(t, expectedInstanceMetrics[metric.Name], metric.Value)
		}

		commonMetrics := <-test.m
		assert.Len(t, commonMetrics.Data.Simplemetrics, len(expectedCommonMetrics))
		for _, metric := range commonMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedCommonMetrics, metric.Name)
			switch metric.Name {
			case "nginx.http.request.count": // delta of sum
				assert.Equal(t, expectedCommonMetrics["nginx.http.request.count"], metric.Value)
			case "nginx.http.request.current": // average
				assert.Equal(t, expectedCommonMetrics["nginx.http.request.current"], metric.Value)
			}
		}

		sslMetrics := <-test.m
		assert.Len(t, sslMetrics.Data.Simplemetrics, len(expectedSSLMetrics))
		for _, metric := range sslMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedSSLMetrics, metric.Name)
			assert.Equal(t, expectedSSLMetrics[metric.Name], metric.Value)
		}

		serverZoneMetrics := <-test.m
		assert.Len(t, serverZoneMetrics.Data.Simplemetrics, len(expectedServerZoneMetrics))
		for _, dimension := range serverZoneMetrics.Data.Dimensions {
			switch dimension.Name {
			case "server_zone":
				assert.Equal(t, serverZoneName, dimension.Value)
			}
		}
		for _, metric := range serverZoneMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedServerZoneMetrics, metric.Name)
			assert.Equal(t, expectedServerZoneMetrics[metric.Name], metric.Value)
		}

		streamServerZoneMetrics := <-test.m
		assert.Len(t, streamServerZoneMetrics.Data.Simplemetrics, len(expectedStreamServerZoneMetrics))
		for _, dimension := range streamServerZoneMetrics.Data.Dimensions {
			switch dimension.Name {
			case "server_zone":
				assert.Equal(t, streamServerZoneName, dimension.Value)
			}
		}
		for _, metric := range streamServerZoneMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedStreamServerZoneMetrics, metric.Name)
			assert.Equal(t, expectedStreamServerZoneMetrics[metric.Name], metric.Value)
		}

		locationZoneMetrics := <-test.m
		assert.Len(t, locationZoneMetrics.Data.Simplemetrics, len(expectedLocationZoneMetrics))
		for _, dimension := range locationZoneMetrics.Data.Dimensions {
			switch dimension.Name {
			case "location_zone":
				assert.Equal(t, locationZoneName, dimension.Value)
			}
		}
		for _, metric := range locationZoneMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedLocationZoneMetrics, metric.Name)
			assert.Equal(t, expectedLocationZoneMetrics[metric.Name], metric.Value)
		}

		slabMetrics := <-test.m
		assert.Len(t, slabMetrics.Data.Simplemetrics, len(expectedSlabMetrics))
		for _, dimension := range slabMetrics.Data.Dimensions {
			switch dimension.Name {
			case "zone":
				assert.Equal(t, serverZoneName, dimension.Value)
			}
		}
		for _, metric := range slabMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedSlabMetrics, metric.Name)
			assert.Equal(t, expectedSlabMetrics[metric.Name], metric.Value)
		}

		slabSlotsMetrics := <-test.m
		assert.Len(t, slabSlotsMetrics.Data.Simplemetrics, len(expectedSlabSlotMetrics))
		for _, metric := range slabSlotsMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedSlabSlotMetrics, metric.Name)
			assert.Equal(t, expectedSlabSlotMetrics[metric.Name], metric.Value)
		}

		limitConnectionsMetrics := <-test.m
		assert.Len(t, limitConnectionsMetrics.Data.Simplemetrics, len(expectedHTTPLimitConnsMetrics))
		for _, dimension := range limitConnectionsMetrics.Data.Dimensions {
			switch dimension.Name {
			case "limit_conn_zone":
				assert.Equal(t, limitConnectionsName, dimension.Value)
			}
		}
		for _, metric := range limitConnectionsMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedHTTPLimitConnsMetrics, metric.Name)
			assert.Equal(t, expectedHTTPLimitConnsMetrics[metric.Name], metric.Value)
		}

		limitRequestsMetrics := <-test.m
		assert.Len(t, limitRequestsMetrics.Data.Simplemetrics, len(expectedHTTPLimitReqsMetrics))
		for _, dimension := range limitRequestsMetrics.Data.Dimensions {
			switch dimension.Name {
			case "limit_req_zone":
				assert.Equal(t, limitRequestName, dimension.Value)
			}
		}
		for _, metric := range limitRequestsMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedHTTPLimitReqsMetrics, metric.Name)
			assert.Equal(t, expectedHTTPLimitReqsMetrics[metric.Name], metric.Value)
		}

		cacheZoneMetrics := <-test.m
		assert.Len(t, cacheZoneMetrics.Data.Simplemetrics, len(expectedCacheZoneMetrics))
		for _, dimension := range cacheZoneMetrics.Data.Dimensions {
			switch dimension.Name {
			case "cache_zone":
				assert.Equal(t, cacheZoneName, dimension.Value)
			}
		}
		for _, metric := range cacheZoneMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedCacheZoneMetrics, metric.Name)
			assert.Equal(t, expectedCacheZoneMetrics[metric.Name], metric.Value)
		}

		httpPeer1upstreamMetrics := <-test.m
		assert.Len(t, httpPeer1upstreamMetrics.Data.Simplemetrics, len(expectedHttpPeer1UpstreamMetrics))
		for _, dimension := range httpPeer1upstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			case "peer.name":
				assert.Equal(t, upstreamPeer1Name, dimension.Value)
			case "peer.address":
				assert.Equal(t, upstreamPeer1ServerAddress, dimension.Value)
			}
		}
		for _, metric := range httpPeer1upstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedHttpPeer1UpstreamMetrics, metric.Name)
			assert.Equal(t, expectedHttpPeer1UpstreamMetrics[metric.Name], metric.Value)
		}

		httpPeer2upstreamMetrics := <-test.m
		assert.Len(t, httpPeer2upstreamMetrics.Data.Simplemetrics, len(expectedHttpPeer2UpstreamMetrics))
		for _, dimension := range httpPeer2upstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			case "peer.name":
				assert.Equal(t, upstreamPeer2Name, dimension.Value)
			case "peer.address":
				assert.Equal(t, upstreamPeer2ServerAddress, dimension.Value)
			}
		}
		for _, metric := range httpPeer2upstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedHttpPeer2UpstreamMetrics, metric.Name)
			assert.Equal(t, expectedHttpPeer2UpstreamMetrics[metric.Name], metric.Value)
		}

		httpUpstreamMetrics := <-test.m
		assert.Len(t, httpUpstreamMetrics.Data.Simplemetrics, len(expectedHttpUpstreamMetrics))
		for _, dimension := range httpUpstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			}
		}
		for _, metric := range httpUpstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedHttpUpstreamMetrics, metric.Name)
			assert.Equal(t, expectedHttpUpstreamMetrics[metric.Name], metric.Value)
		}

		streamPeer1upstreamMetrics := <-test.m
		assert.Len(t, streamPeer1upstreamMetrics.Data.Simplemetrics, len(expectedStreamPeer1UpstreamMetrics))
		for _, dimension := range streamPeer1upstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			case "peer.name":
				assert.Equal(t, streamUpstreamPeer1Name, dimension.Value)
			case "peer.address":
				assert.Equal(t, streamUpstreamPeer1ServerAddress, dimension.Value)
			}
		}

		streamPeer2upstreamMetrics := <-test.m
		assert.Len(t, streamPeer2upstreamMetrics.Data.Simplemetrics, len(expectedStreamPeer2UpstreamMetrics))
		for _, dimension := range streamPeer2upstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			case "peer.name":
				assert.Equal(t, streamUpstreamPeer2Name, dimension.Value)
			case "peer.address":
				assert.Equal(t, streamUpstreamPeer2ServerAddress, dimension.Value)
			}
		}

		for _, metric := range streamPeer1upstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedStreamPeer1UpstreamMetrics, metric.Name)
			assert.Equal(t, expectedStreamPeer1UpstreamMetrics[metric.Name], metric.Value)
		}

		for _, metric := range streamPeer2upstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedStreamPeer2UpstreamMetrics, metric.Name)
			assert.Equal(t, expectedStreamPeer2UpstreamMetrics[metric.Name], metric.Value)
		}

		streamUpstreamMetrics := <-test.m
		assert.Len(t, streamUpstreamMetrics.Data.Simplemetrics, len(expectedStreamUpstreamMetrics))
		for _, dimension := range streamUpstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			}
		}
		for _, metric := range streamUpstreamMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedStreamUpstreamMetrics, metric.Name)
			assert.Equal(t, expectedStreamUpstreamMetrics[metric.Name], metric.Value)
		}

		workerMetrics := <-test.m
		assert.Len(t, workerMetrics.Data.Simplemetrics, len(expectedWorkerMetrics))

		for _, dimension := range streamUpstreamMetrics.Data.Dimensions {
			switch dimension.Name {
			case "process_id":
				assert.Equal(t, workerProcessID, dimension.Value)
			}
		}

		for _, metric := range workerMetrics.Data.Simplemetrics {
			assert.Contains(t, expectedWorkerMetrics, metric.Name)
			assert.Equal(t, expectedWorkerMetrics[metric.Name], metric.Value)
		}

		var extraMetrics []*metrics.StatsEntityWrapper
	EMWAIT:
		for {
			select {
			case <-ctx.Done():
				break EMWAIT
			case em := <-test.m:
				extraMetrics = append(extraMetrics, em)
			default:
				break EMWAIT
			}
		}

		assert.Len(t, extraMetrics, 0, "metrics returned but not tested")
	}
}

func TestGetLatestAPIVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/api":
			data := []byte("[1,2,3,4,5,6,7,8,9]")
			_, err := rw.Write(data)
			require.NoError(t, err)
		case "/oldapi":
			data := []byte("[1,2,3,4,5]")
			_, err := rw.Write(data)
			require.NoError(t, err)
		case "/api25":
			data := []byte("[1,2,3,4,5,6,7]")
			_, err := rw.Write(data)
			require.NoError(t, err)
		default:
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("")
			_, err := rw.Write(data)
			require.NoError(t, err)
		}
	}))
	defer server.Close()

	tests := []struct {
		name               string
		clientVersion      int
		endpoint           string
		expectedAPIVersion int
		expectErr          bool
	}{
		{
			name:               "NGINX Plus R30",
			clientVersion:      7,
			endpoint:           "/api",
			expectedAPIVersion: 9,
			expectErr:          false,
		},
		{
			name:               "NGINX Plus R25",
			clientVersion:      7,
			endpoint:           "/api25",
			expectedAPIVersion: 7,
			expectErr:          false,
		},
		{
			name:               "old nginx plus",
			clientVersion:      7,
			endpoint:           "/oldapi",
			expectedAPIVersion: 0,
			expectErr:          true,
		},
		{
			name:               "invalid path",
			clientVersion:      7,
			endpoint:           "/notexisting",
			expectedAPIVersion: 0,
			expectErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &NginxPlus{
				clientVersion: tt.clientVersion,
			}
			got, err := c.getLatestAPIVersion(context.Background(), fmt.Sprintf("%s%s", server.URL, tt.endpoint))
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedAPIVersion, got)
		})
	}
}
