/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	"github.com/stretchr/testify/assert"
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
)

type FakeNginxPlus struct {
	*NginxPlus
}

// Collect is fake collector that hard codes a stats struct response to avoid dependency on external NGINX Plus api
func (f *FakeNginxPlus) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
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
						ID:       0,
						Server:   upstreamPeer1ServerAddress,
						Name:     upstreamPeer1Name,
						Backup:   false,
						Weight:   1,
						State:    "up",
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
						ID:       1,
						Server:   upstreamPeer2ServerAddress,
						Name:     upstreamPeer2Name,
						Backup:   false,
						Weight:   1,
						State:    "up",
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
						ID:       0,
						Server:   upstreamPeer2ServerAddress,
						Name:     upstreamPeer2Name,
						Backup:   false,
						Weight:   1,
						State:    "up",
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
		"plus.http.request.count":      currentZoneRequests - previousZoneRequests,
		"plus.http.response.count":     currentZoneResponses - previousZoneResponses,
		"plus.http.status.discarded":   0,
		"plus.http.status.processing":  0,
		"plus.http.request.bytes_rcvd": currentZoneReceived - previousZoneReceived,
		"plus.http.request.bytes_sent": currentZoneSent - previousZoneSent,
		"plus.http.status.1xx":         0,
		"plus.http.status.2xx":         currentZoneResponses - previousZoneResponses,
		"plus.http.status.3xx":         0,
		"plus.http.status.4xx":         0,
		"plus.http.status.5xx":         0,
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
		"plus.http.upstream.peers.request.count":           currentZoneRequests,
		"plus.http.upstream.peers.response.count":          currentZoneResponses,
		"plus.http.upstream.peers.status.1xx":              0,
		"plus.http.upstream.peers.status.2xx":              currentZoneResponses,
		"plus.http.upstream.peers.status.3xx":              0,
		"plus.http.upstream.peers.status.4xx":              0,
		"plus.http.upstream.peers.status.5xx":              0,
		"plus.http.upstream.peers.bytes_sent":              currentZoneSent,
		"plus.http.upstream.peers.bytes_rcvd":              currentZoneReceived,
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

	hostInfo := &proto.HostInfo{
		Hostname: "MyServer",
	}
	tests := []struct {
		baseDimensions *metrics.CommonDim
		m              chan *proto.StatsEntity
	}{
		{
			baseDimensions: metrics.NewCommonDim(hostInfo, &config.Config{}, ""),
			m:              make(chan *proto.StatsEntity, 127),
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
		assert.Len(t, instanceMetrics.Simplemetrics, len(expectedInstanceMetrics))
		for _, metric := range instanceMetrics.Simplemetrics {
			assert.Contains(t, expectedInstanceMetrics, metric.Name)
			switch metric.Name {
			case "nginx.status":
				assert.Equal(t, expectedInstanceMetrics["nginx.status"], metric.Value)
			case "nginx.config.generation":
				assert.Equal(t, expectedInstanceMetrics["nginx.config.generation"], metric.Value)
			}
		}

		commonMetrics := <-test.m
		assert.Len(t, commonMetrics.Simplemetrics, len(expectedCommonMetrics))
		for _, metric := range commonMetrics.Simplemetrics {
			assert.Contains(t, expectedCommonMetrics, metric.Name)
			switch metric.Name {
			case "nginx.http.request.count": // delta of sum
				assert.Equal(t, expectedCommonMetrics["nginx.http.request.count"], metric.Value)
			case "nginx.http.request.current": // average
				assert.Equal(t, expectedCommonMetrics["nginx.http.request.current"], metric.Value)
			}
		}

		sslMetrics := <-test.m
		assert.Len(t, sslMetrics.Simplemetrics, len(expectedSSLMetrics))
		for _, metric := range sslMetrics.Simplemetrics {
			assert.Contains(t, expectedSSLMetrics, metric.Name)
		}

		serverZoneMetrics := <-test.m
		assert.Len(t, serverZoneMetrics.Simplemetrics, len(expectedServerZoneMetrics))
		for _, dimension := range serverZoneMetrics.Dimensions {
			switch dimension.Name {
			case "server_zone":
				assert.Equal(t, serverZoneName, dimension.Value)
			}
		}
		for _, metric := range serverZoneMetrics.Simplemetrics {
			assert.Contains(t, expectedServerZoneMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.request.count":
				assert.Equal(t, expectedServerZoneMetrics["plus.http.request.count"], metric.Value)
			case "plus.http.response.count":
				assert.Equal(t, expectedServerZoneMetrics["plus.http.response.count"], metric.Value)
			case "plus.http.status.2xx":
				assert.Equal(t, expectedServerZoneMetrics["plus.http.status.2xx"], metric.Value)
			case "plus.http.request.bytes_rcvd":
				assert.Equal(t, expectedServerZoneMetrics["plus.http.request.bytes_rcvd"], metric.Value)
			case "plus.http.request.bytes_sent":
				assert.Equal(t, expectedServerZoneMetrics["plus.http.request.bytes_sent"], metric.Value)
			}
		}

		streamServerZoneMetrics := <-test.m
		assert.Len(t, streamServerZoneMetrics.Simplemetrics, len(expectedStreamServerZoneMetrics))
		for _, dimension := range streamServerZoneMetrics.Dimensions {
			switch dimension.Name {
			case "server_zone":
				assert.Equal(t, streamServerZoneName, dimension.Value)
			}
		}
		for _, metric := range streamServerZoneMetrics.Simplemetrics {
			assert.Contains(t, expectedStreamServerZoneMetrics, metric.Name)
			switch metric.Name {
			case "plus.stream.connections":
				assert.Equal(t, expectedStreamServerZoneMetrics["plus.stream.connections"], metric.Value)
			case "plus.stream.status.total":
				assert.Equal(t, expectedStreamServerZoneMetrics["plus.stream.status.total"], metric.Value)
			case "plus.stream.status.2xx":
				assert.Equal(t, expectedStreamServerZoneMetrics["plus.stream.status.2xx"], metric.Value)
			case "plus.stream.bytes_rcvd":
				assert.Equal(t, expectedStreamServerZoneMetrics["plus.stream.bytes_rcvd"], metric.Value)
			case "plus.stream.bytes_sent":
				assert.Equal(t, expectedStreamServerZoneMetrics["plus.stream.bytes_sent"], metric.Value)
			}
		}

		locationZoneMetrics := <-test.m
		assert.Len(t, locationZoneMetrics.Simplemetrics, len(expectedLocationZoneMetrics))
		for _, dimension := range locationZoneMetrics.Dimensions {
			switch dimension.Name {
			case "location_zone":
				assert.Equal(t, locationZoneName, dimension.Value)
			}
		}
		for _, metric := range locationZoneMetrics.Simplemetrics {
			assert.Contains(t, expectedLocationZoneMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.request.count":
				assert.Equal(t, expectedLocationZoneMetrics["plus.http.request.count"], metric.Value)
			case "plus.http.response.count":
				assert.Equal(t, expectedLocationZoneMetrics["plus.http.response.count"], metric.Value)
			case "plus.http.status.2xx":
				assert.Equal(t, expectedLocationZoneMetrics["plus.http.status.2xx"], metric.Value)
			case "plus.http.request.bytes_rcvd":
				assert.Equal(t, expectedLocationZoneMetrics["plus.http.request.bytes_rcvd"], metric.Value)
			case "plus.http.request.bytes_sent":
				assert.Equal(t, expectedLocationZoneMetrics["plus.http.request.bytes_sent"], metric.Value)
			}
		}

		cacheZoneMetrics := <-test.m
		assert.Len(t, cacheZoneMetrics.Simplemetrics, len(expectedCacheZoneMetrics))
		for _, dimension := range cacheZoneMetrics.Dimensions {
			switch dimension.Name {
			case "cache_zone":
				assert.Equal(t, cacheZoneName, dimension.Value)
			}
		}
		for _, metric := range cacheZoneMetrics.Simplemetrics {
			assert.Contains(t, expectedCacheZoneMetrics, metric.Name)
			switch metric.Name {
			case "plus.cache.size":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.size"], metric.Value)
			case "plus.cache.max_size":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.max_size"], metric.Value)
			case "plus.cache.hit.responses":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.hit.responses"], metric.Value)
			case "plus.cache.hit.bytes":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.hit.bytes"], metric.Value)
			case "plus.cache.miss.responses":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.miss.responses"], metric.Value)
			case "plus.cache.miss.bytes":
				assert.Equal(t, expectedCacheZoneMetrics["plus.cache.miss.bytes"], metric.Value)
			}
		}

		httpPeer1upstreamMetrics := <-test.m
		assert.Len(t, httpPeer1upstreamMetrics.Simplemetrics, len(expectedHttpPeer1UpstreamMetrics))
		for _, dimension := range httpPeer1upstreamMetrics.Dimensions {
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
		for _, metric := range httpPeer1upstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedHttpPeer1UpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.upstream.peers.header_time":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.header_time"], metric.Value)
			case "plus.http.upstream.peers.response.time":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.response.time"], metric.Value)
			case "plus.http.upstream.peers.request.count":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.request.count"], metric.Value)
			case "plus.http.upstream.peers.response.count":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.response.count"], metric.Value)
			case "plus.http.upstream.peers.status.2xx":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.status.2xx"], metric.Value)
			case "plus.http.upstream.peers.bytes_sent":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.bytes_sent"], metric.Value)
			case "plus.http.upstream.peers.bytes_rcvd":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.bytes_rcvd"], metric.Value)
			case "plus.http.upstream.peers.state.up":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.state.up"], metric.Value)
			}
		}

		httpPeer2upstreamMetrics := <-test.m
		assert.Len(t, httpPeer2upstreamMetrics.Simplemetrics, len(expectedHttpPeer2UpstreamMetrics))
		for _, dimension := range httpPeer2upstreamMetrics.Dimensions {
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
		for _, metric := range httpPeer2upstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedHttpPeer2UpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.upstream.peers.header_time":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.header_time"], metric.Value)
			case "plus.http.upstream.peers.response.time":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.response.time"], metric.Value)
			case "plus.http.upstream.peers.request.count":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.request.count"], metric.Value)
			case "plus.http.upstream.peers.response.count":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.response.count"], metric.Value)
			case "plus.http.upstream.peers.status.2xx":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.status.2xx"], metric.Value)
			case "plus.http.upstream.peers.bytes_sent":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.bytes_sent"], metric.Value)
			case "plus.http.upstream.peers.bytes_rcvd":
				assert.Equal(t, expectedHttpPeer2UpstreamMetrics["plus.http.upstream.peers.bytes_rcvd"], metric.Value)
			case "plus.http.upstream.peers.state.up":
				assert.Equal(t, expectedHttpPeer1UpstreamMetrics["plus.http.upstream.peers.state.up"], metric.Value)
			}
		}

		httpUpstreamMetrics := <-test.m
		assert.Len(t, httpUpstreamMetrics.Simplemetrics, len(expectedHttpUpstreamMetrics))
		for _, dimension := range httpUpstreamMetrics.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			}
		}
		for _, metric := range httpUpstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedHttpUpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.upstream.queue.maxsize":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.queue.maxsize"], metric.Value)
			case "plus.http.upstream.zombies":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.zombies"], metric.Value)
			case "plus.http.upstream.peers.count":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.count"], metric.Value)
			case "plus.http.upstream.peers.total.up":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.total.up"], metric.Value)
			case "plus.http.upstream.peers.header_time.count":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.header_time.count"], metric.Value)
			case "plus.http.upstream.peers.header_time.max":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.header_time.max"], metric.Value)
			case "plus.http.upstream.peers.header_time.median":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.header_time.median"], metric.Value)
			case "plus.http.upstream.peers.header_time.pctl95":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.header_time.pctl95"], metric.Value)
			case "plus.http.upstream.peers.response.time.count":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.response.time.count"], metric.Value)
			case "plus.http.upstream.peers.response.time.max":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.response.time.max"], metric.Value)
			case "plus.http.upstream.peers.response.time.median":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.response.time.median"], metric.Value)
			case "plus.http.upstream.peers.response.time.pctl95":
				assert.Equal(t, expectedHttpUpstreamMetrics["plus.http.upstream.peers.response.time.pctl95"], metric.Value)
			}
		}

		streamPeer1upstreamMetrics := <-test.m
		assert.Len(t, streamPeer1upstreamMetrics.Simplemetrics, len(expectedStreamPeer1UpstreamMetrics))
		for _, dimension := range streamPeer1upstreamMetrics.Dimensions {
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
		assert.Len(t, streamPeer2upstreamMetrics.Simplemetrics, len(expectedStreamPeer2UpstreamMetrics))
		for _, dimension := range streamPeer2upstreamMetrics.Dimensions {
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

		for _, metric := range streamPeer1upstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedStreamPeer1UpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.stream.upstream.peers.conn.active":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.conn.active"], metric.Value)
			case "plus.stream.upstream.peers.conn.count":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.conn.count"], metric.Value)
			case "plus.stream.upstream.peers.connect_time":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.connect_time"], metric.Value)
			case "plus.stream.upstream.peers.ttfb":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.ttfb"], metric.Value)
			case "plus.stream.upstream.peers.response.time":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.response.time"], metric.Value)
			case "plus.stream.upstream.peers.bytes_sent":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.bytes_sent"], metric.Value)
			case "plus.stream.upstream.peers.bytes_rcvd":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.bytes_rcvd"], metric.Value)
			case "plus.stream.upstream.peers.state.up":
				assert.Equal(t, expectedStreamPeer1UpstreamMetrics["plus.stream.upstream.peers.state.up"], metric.Value)
			}
		}

		for _, metric := range streamPeer2upstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedStreamPeer2UpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.stream.upstream.peers.conn.active":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.conn.active"], metric.Value)
			case "plus.stream.upstream.peers.conn.count":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.conn.count"], metric.Value)
			case "plus.stream.upstream.peers.connect_time":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.connect_time"], metric.Value)
			case "plus.stream.upstream.peers.ttfb":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.ttfb"], metric.Value)
			case "plus.stream.upstream.peers.response.time":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.response.time"], metric.Value)
			case "plus.stream.upstream.peers.bytes_sent":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.bytes_sent"], metric.Value)
			case "plus.stream.upstream.peers.bytes_rcvd":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.bytes_rcvd"], metric.Value)
			case "plus.stream.upstream.peers.state.up":
				assert.Equal(t, expectedStreamPeer2UpstreamMetrics["plus.stream.upstream.peers.state.up"], metric.Value)
			}
		}

		streamUpstreamMetrics := <-test.m
		assert.Len(t, streamUpstreamMetrics.Simplemetrics, len(expectedStreamUpstreamMetrics))
		for _, dimension := range streamUpstreamMetrics.Dimensions {
			switch dimension.Name {
			case "upstream":
				assert.Equal(t, upstreamName, dimension.Value)
			case "upstream_zone":
				assert.Equal(t, upstreamZoneName, dimension.Value)
			}
		}
		for _, metric := range streamUpstreamMetrics.Simplemetrics {
			assert.Contains(t, expectedStreamUpstreamMetrics, metric.Name)
			switch metric.Name {
			case "plus.stream.upstream.zombies":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.zombies"], metric.Value)
			case "plus.stream.upstream.peers.count":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.count"], metric.Value)
			case "plus.stream.upstream.peers.total.up":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.total.up"], metric.Value)
			case "plus.stream.upstream.peers.response.time.count":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.response.time.count"], metric.Value)
			case "plus.stream.upstream.peers.response.time.max":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.response.time.max"], metric.Value)
			case "plus.stream.upstream.peers.response.time.median":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.response.time.median"], metric.Value)
			case "plus.stream.upstream.peers.response.time.pctl95":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.response.time.pctl95"], metric.Value)
			case "plus.stream.upstream.peers.connect_time.count":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.connect_time.count"], metric.Value)
			case "plus.stream.upstream.peers.connect_time.max":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.connect_time.max"], metric.Value)
			case "plus.stream.upstream.peers.connect_time.median":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.connect_time.median"], metric.Value)
			case "plus.stream.upstream.peers.connect_time.pctl95":
				assert.Equal(t, expectedStreamUpstreamMetrics["plus.stream.upstream.peers.connect_time.pctl95"], metric.Value)
			}
		}

		slabMetrics := <-test.m
		assert.Len(t, slabMetrics.Simplemetrics, len(expectedSlabMetrics))
		for _, dimension := range slabMetrics.Dimensions {
			switch dimension.Name {
			case "zone":
				assert.Equal(t, serverZoneName, dimension.Value)
			}
		}
		for _, metric := range slabMetrics.Simplemetrics {
			assert.Contains(t, expectedSlabMetrics, metric.Name)
			switch metric.Name {
			case "plus.slab.pages.used":
				assert.Equal(t, expectedSlabMetrics["plus.slab.pages.used"], metric.Value)
			case "plus.slab.pages.free":
				assert.Equal(t, expectedSlabMetrics["plus.slab.pages.free"], metric.Value)
			case "plus.slab.pages.total":
				assert.Equal(t, expectedSlabMetrics["plus.slab.pages.total"], metric.Value)
			case "plus.slab.pages.pct_used":
				assert.Equal(t, expectedSlabMetrics["plus.slab.pages.pct_used"], metric.Value)
			}
		}

		slabSlotsMetrics := <-test.m
		assert.Len(t, slabSlotsMetrics.Simplemetrics, len(expectedSlabSlotMetrics))
		for _, metric := range slabSlotsMetrics.Simplemetrics {
			assert.Contains(t, expectedSlabSlotMetrics, metric.Name)
			assert.Equal(t, expectedSlabSlotMetrics[metric.Name], metric.Value)
		}

		limitConnectionsMetrics := <-test.m
		assert.Len(t, limitConnectionsMetrics.Simplemetrics, len(expectedHTTPLimitConnsMetrics))
		for _, dimension := range limitConnectionsMetrics.Dimensions {
			switch dimension.Name {
			case "limit_conn_zone":
				assert.Equal(t, limitConnectionsName, dimension.Value)
			}
		}
		for _, metric := range limitConnectionsMetrics.Simplemetrics {
			assert.Contains(t, expectedHTTPLimitConnsMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.limit_conns.passed":
				assert.Equal(t, expectedHTTPLimitConnsMetrics["plus.http.limit_conns.passed"], metric.Value)
			case "plus.http.limit_conns.rejected":
				assert.Equal(t, expectedHTTPLimitConnsMetrics["plus.http.limit_conns.rejected"], metric.Value)
			case "plus.http.limit_conns.rejected_dry_run":
				assert.Equal(t, expectedHTTPLimitConnsMetrics["plus.http.limit_conns.rejected_dry_run"], metric.Value)
			}
		}

		limitRequestsMetrics := <-test.m
		assert.Len(t, limitRequestsMetrics.Simplemetrics, len(expectedHTTPLimitReqsMetrics))
		for _, dimension := range limitRequestsMetrics.Dimensions {
			switch dimension.Name {
			case "limit_req_zone":
				assert.Equal(t, limitRequestName, dimension.Value)

			}
		}
		for _, metric := range limitRequestsMetrics.Simplemetrics {
			assert.Contains(t, expectedHTTPLimitReqsMetrics, metric.Name)
			switch metric.Name {
			case "plus.http.limit_reqs.passed":
				assert.Equal(t, expectedHTTPLimitReqsMetrics["plus.http.limit_reqs.passed"], metric.Value)
			case "plus.http.limit_reqs.delayed":
				assert.Equal(t, expectedHTTPLimitReqsMetrics["plus.http.limit_reqs.delayed"], metric.Value)
			case "plus.http.limit_reqs.rejected":
				assert.Equal(t, expectedHTTPLimitReqsMetrics["plus.http.limit_reqs.rejected"], metric.Value)
			case "plus.http.limit_reqs.delayed_dry_run":
				assert.Equal(t, expectedHTTPLimitReqsMetrics["plus.http.limit_reqs.delayed_dry_run"], metric.Value)
			case "plus.http.limit_reqs.rejected_dry_run":
				assert.Equal(t, expectedHTTPLimitReqsMetrics["plus.http.limit_reqs.rejected_dry_run"], metric.Value)
			}
		}

		var extraMetrics []*proto.StatsEntity
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
