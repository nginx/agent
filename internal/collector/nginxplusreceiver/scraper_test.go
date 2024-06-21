// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const endpointRootPath = "/api/9/"

func TestScraper(t *testing.T) {
	nginxPlusMock := newMockServer(t)
	defer nginxPlusMock.Close()

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = nginxPlusMock.URL + "/api"
	require.NoError(t, component.ValidateConfig(cfg))

	scraper, err := newNginxPlusScraper(receivertest.NewNopSettings(), cfg)
	require.NoError(t, err)

	actualMetrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreTimestamp(),
		pmetrictest.IgnoreMetricsOrder()))
}

func newMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("got path %s\n", req.URL.Path)

		var payload string
		switch req.URL.Path {
		case "/api/":
			payload = "[1,2,3,4,5,6,7,8,9]"
		case endpointRootPath:
			payload = `[
				"nginx",
				"processes",
				"connections",
				"slabs",
				"http",
				"stream",
				"resolvers",
				"ssl",
				"workers"
			]`
		case endpointRootPath + "nginx":
			payload = `{
				"version": "1.25.3",
				"build": "nginx-plus-r31-p2",
				"address": "172.20.0.2",
				"generation": 1,
				"load_timestamp": "2024-06-21T07:30:13.053Z",
				"timestamp": "2024-06-21T07:38:58.900Z",
				"pid": 10,
				"ppid": 1
			}`
		case endpointRootPath + "http/caches":
			payload = `{
				"http_cache": {
					"size": 0,
					"max_size": 104857600,
					"cold": false,
					"hit": {
						"responses": 0,
						"bytes": 0
					},
					"stale": {
						"responses": 0,
						"bytes": 0
					},
					"updating": {
						"responses": 0,
						"bytes": 0
					},
					"revalidated": {
						"responses": 0,
						"bytes": 0
					},
					"miss": {
						"responses": 0,
						"bytes": 0,
						"responses_written": 0,
						"bytes_written": 0
					},
					"expired": {
						"responses": 0,
						"bytes": 0,
						"responses_written": 0,
						"bytes_written": 0
					},
					"bypass": {
						"responses": 0,
						"bytes": 0,
						"responses_written": 0,
						"bytes_written": 0
					}
				}
			}`
		case endpointRootPath + "processes":
			payload = `{"respawned": 0}`
		case endpointRootPath + "slabs":
			payload = `{
				"zone_one": {
					"pages": {
					"used": 2,
					"free": 5
					},
					"slots": {
					"8": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"16": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"32": {
						"used": 1,
						"free": 126,
						"reqs": 1,
						"fails": 0
					},
					"64": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"128": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"256": {
						"used": 1,
						"free": 15,
						"reqs": 1,
						"fails": 0
					},
					"512": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"1024": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"2048": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					}
					}
				},
				"test": {
					"pages": {
					"used": 4,
					"free": 11
					},
					"slots": {
					"8": {
						"used": 2,
						"free": 502,
						"reqs": 2,
						"fails": 0
					},
					"16": {
						"used": 1,
						"free": 253,
						"reqs": 1,
						"fails": 0
					},
					"32": {
						"used": 1,
						"free": 126,
						"reqs": 1,
						"fails": 0
					},
					"64": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"128": {
						"used": 2,
						"free": 30,
						"reqs": 2,
						"fails": 0
					},
					"256": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"512": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"1024": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"2048": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					}
					}
				},
				"http_cache": {
					"pages": {
					"used": 2,
					"free": 2542
					},
					"slots": {
					"8": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"16": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"32": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"64": {
						"used": 1,
						"free": 63,
						"reqs": 1,
						"fails": 0
					},
					"128": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"256": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"512": {
						"used": 1,
						"free": 7,
						"reqs": 1,
						"fails": 0
					},
					"1024": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					},
					"2048": {
						"used": 0,
						"free": 0,
						"reqs": 0,
						"fails": 0
					}
					}
				}
			}`
		case endpointRootPath + "connections":
			payload = `{
				"accepted": 11,
				"dropped": 0,
				"active": 2,
				"idle": 0
			}`
		case endpointRootPath + "http/requests":
			payload = `{
				"total": 47,
				"current": 1
			}`
		case endpointRootPath + "ssl":
			payload = `{
				"handshakes": 0,
				"session_reuses": 0,
				"handshakes_failed": 0,
				"no_common_protocol": 0,
				"no_common_cipher": 0,
				"handshake_timeout": 0,
				"peer_rejected_cert": 0,
				"verify_failures": {
					"no_cert": 0,
					"expired_cert": 0,
					"revoked_cert": 0,
					"hostname_mismatch": 0,
					"other": 0
				}
			}`
		case endpointRootPath + "http/server_zones":
			payload = `{
				"test": {
					"processing": 1,
					"requests": 32,
					"responses": {
					"1xx": 0,
					"2xx": 29,
					"3xx": 0,
					"4xx": 2,
					"5xx": 0,
					"codes": {
						"200": 29,
						"404": 2
					},
					"total": 31
					},
					"discarded": 0,
					"received": 3312,
					"sent": 21860
				}
			}`
		case endpointRootPath + "http/upstreams":
			payload = `{
				"test": {
					"peers": [],
					"keepalive": 0,
					"zombies": 0,
					"zone": "test"
				},
				"test-drain": {
					"peers": [
					{
						"id": 0,
						"server": "127.0.0.1:9001",
						"name": "127.0.0.1:9001",
						"backup": false,
						"weight": 1,
						"state": "draining",
						"active": 0,
						"requests": 0,
						"responses": {
						"1xx": 0,
						"2xx": 0,
						"3xx": 0,
						"4xx": 0,
						"5xx": 0,
						"codes": {},
						"total": 0
						},
						"sent": 0,
						"received": 0,
						"fails": 0,
						"unavail": 0,
						"health_checks": {
						"checks": 0,
						"fails": 0,
						"unhealthy": 0
						},
						"downtime": 0
					}
					],
					"keepalive": 0,
					"zombies": 0,
					"zone": "test-drain"
				}
			}`
		case endpointRootPath + "http/location_zones":
			payload = `{
				"location_test": {
					"requests": 34,
					"responses": {
					"1xx": 0,
					"2xx": 31,
					"3xx": 0,
					"4xx": 3,
					"5xx": 0,
					"codes": {
						"200": 31,
						"404": 3
					},
					"total": 34
					},
					"discarded": 0,
					"received": 3609,
					"sent": 23265
				}
			}`
		case endpointRootPath + "resolvers":
			payload = `{
				"resolver_test": {
					"requests": {
						"name": 481,
						"srv": 0,
						"addr": 0
					},
					"responses": {
						"noerror": 481,
						"formerr": 0,
						"servfail": 0,
						"nxdomain": 0,
						"notimp": 0,
						"refused": 0,
						"timedout": 0,
						"unknown": 0
					}
				}
			}`
		case endpointRootPath + "http/limit_reqs":
			payload = `{
				"one": {
					"passed": 27,
					"delayed": 10,
					"rejected": 0,
					"delayed_dry_run": 0,
					"rejected_dry_run": 0
				}
			}`
		case endpointRootPath + "http/limit_conns":
			payload = `{
				"addr": {
					"passed": 38,
					"rejected": 0,
					"rejected_dry_run": 0
				}
			}`
		case endpointRootPath + "workers":
			payload = `[
				{
					"id": 0,
					"pid": 17,
					"connections": {
					"accepted": 14,
					"dropped": 0,
					"active": 1,
					"idle": 0
					},
					"http": {
					"requests": {
						"total": 32,
						"current": 1
					}
					}
				},
				{
					"id": 1,
					"pid": 8,
					"connections": {
					"accepted": 2,
					"dropped": 0,
					"active": 1,
					"idle": 0
					},
					"http": {
					"requests": {
						"total": 1,
						"current": 0
					}
					}
				},
				{
					"id": 2,
					"pid": 9,
					"connections": {
					"accepted": 1,
					"dropped": 0,
					"active": 0,
					"idle": 0
					},
					"http": {
					"requests": {
						"total": 1,
						"current": 0
					}
					}
				}
			]`
		case endpointRootPath + "stream":
			payload = `[
				"server_zones",
				"limit_conns",
				"keyvals",
				"zone_sync",
				"upstreams"
			]`
		case endpointRootPath + "stream/server_zones":
			payload = `{
				"stream_test": {
					"processing": 0,
					"connections": 0,
					"sessions": {
						"2xx": 0,
						"4xx": 0,
						"5xx": 0,
						"total": 0
					},
					"discarded": 0,
					"received": 0,
					"sent": 0
				}
			}`
		case endpointRootPath + "stream/upstreams":
			payload = `{
				"stream_test": {
					"peers": [],
					"zombies": 0,
					"zone": "stream_test"
				}
			}`
		case endpointRootPath + "stream/limit_conns":
			payload = `{
				"addr_stream": {
					"passed": 2,
					"rejected": 0,
					"rejected_dry_run": 0
				}
			}`
		case endpointRootPath + "stream/zone_sync":
			payload = `{
				"status": {
					"nodes_online": 1,
					"msgs_in": 3,
					"msgs_out": 0,
					"bytes_in": 116,
					"bytes_out": 0
				},
				"zones": {
					"zone_test_sync": {
					"records_total": 1,
					"records_pending": 0
					}
				}
			}`
		default:
			// no-op
		}

		if payload == "" {
			rw.WriteHeader(404)
		} else {
			rw.WriteHeader(200)
			_, err := rw.Write([]byte(payload))
			require.NoError(t, err)
			return
		}
	}))
}
