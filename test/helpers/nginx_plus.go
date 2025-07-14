// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nginxinc/nginx-plus-go-client/v2/client"

	"github.com/stretchr/testify/assert"
)

const (
	endpointRootPath = "/api/9/"
	serverID         = 1234
)

//nolint:gocyclo,revive,cyclop,maintidx
func NewMockNGINXPlusAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
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
					"size": 432,
					"max_size": 104857600,
					"cold": false,
					"hit": {
						"responses": 11,
						"bytes": 432
					},
					"stale": {
						"responses": 21,
						"bytes": 421
					},
					"updating": {
						"responses": 2,
						"bytes": 324
					},
					"revalidated": {
						"responses": 56,
						"bytes": 42142
					},
					"miss": {
						"responses": 421,
						"bytes": 44,
						"responses_written": 0,
						"bytes_written": 0
					},
					"expired": {
						"responses": 24,
						"bytes": 10,
						"responses_written": 0,
						"bytes_written": 0
					},
					"bypass": {
						"responses": 76,
						"bytes": 244,
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
						"used": 4,
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
						}
					}
				},
				"test": {
					"pages": {
						"used": 3,
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
							"reqs": 52,
							"fails": 22
						},
						"16": {
							"used": 0,
							"free": 0,
							"reqs": 11,
							"fails": 44
						}
					}
				}
			}`
		case endpointRootPath + "connections":
			payload = `{
				"accepted": 11,
				"dropped": 5,
				"active": 2,
				"idle": 1
			}`
		case endpointRootPath + "http/requests":
			payload = `{
				"total": 47,
				"current": 3
			}`
		case endpointRootPath + "ssl":
			payload = `{
				"handshakes": 45,
				"session_reuses": 6,
				"handshakes_failed": 7,
				"no_common_protocol": 2,
				"no_common_cipher": 3,
				"handshake_timeout": 4,
				"peer_rejected_cert": 5,
				"verify_failures": {
					"no_cert": 2,
					"expired_cert": 3,
					"revoked_cert": 4,
					"hostname_mismatch": 5,
					"other": 6
				}
			}`
		case endpointRootPath + "http/server_zones":
			payload = `{
				"test": {
					"processing": 1,
					"requests": 32,
					"responses": {
					"1xx": 5,
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
					"keepalive": 3,
					"zombies": 56,
					"zone": "test-zone"
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
						"active": 879,
						"requests": 432,
						"responses": {
							"1xx": 2,
							"2xx": 3,
							"3xx": 4,
							"4xx": 5,
							"5xx": 6,
							"codes": {},
							"total": 32
						},
						"sent": 121,
						"received": 432,
						"fails": 87,
						"unavail": 99,
        				"header_time" : 20,
        				"response_time" : 36,
						"health_checks": {
							"checks": 321,
							"fails": 22,
							"unhealthy": 11
						},
						"downtime": 0
					}
					],
					"keepalive": 43,
					"zombies": 11,
					"zone": "test-drain-zone",
					"queue": {
						"size": 23412,
						"max_size": 324324,
						"overflows": 321
					}
				}
			}`
		case endpointRootPath + "http/location_zones":
			payload = `{
				"location_test": {
					"requests": 34,
					"responses": {
					"1xx": 7,
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
					"rejected": 22,
					"delayed_dry_run": 10,
					"rejected_dry_run": 3
				}
			}`
		case endpointRootPath + "http/limit_conns":
			payload = `{
				"addr": {
					"passed": 38,
					"rejected": 43,
					"rejected_dry_run": 22
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
					"processing": 44,
					"connections": 22,
					"sessions": {
						"2xx": 1,
						"4xx": 2,
						"5xx": 3,
						"total": 6
					},
					"discarded": 43,
					"received": 22,
					"sent": 11
				}
			}`
		case endpointRootPath + "stream/upstreams":
			payload = `{
				"upstream_test": {
					"peers": [
						{
							"id" : 0,
							"server" : "10.0.0.1:12347",
							"name" : "10.0.0.1:12347",
							"backup" : false,
							"weight" : 5,
							"state" : "up",
							"active" : 22,
							"ssl" : {
								"handshakes" : 200,
								"handshakes_failed" : 4,
								"session_reuses" : 189,
								"no_common_protocol" : 4,
								"handshake_timeout" : 0,
								"peer_rejected_cert" : 0,
								"verify_failures" : {
									"expired_cert" : 2,
									"revoked_cert" : 1,
									"hostname_mismatch" : 2,
									"other" : 1
								}
							},
							"max_conns" : 50,
							"connections" : 667231,
							"connect_time" : 432,
							"response_time" : 54325,
							"first_byte_time" : 4322,
							"sent" : 251946292,
							"received" : 19222475454,
							"fails" : 43,
							"unavail" : 45,
							"health_checks" : {
								"checks" : 26214,
								"fails" : 23,
								"unhealthy" : 2,
								"last_passed" : true
							},
							"downtime" : 0,
							"downstart" : "2022-06-28T11:09:21.602Z",
							"selected" : "2022-06-28T15:01:25.000Z"
						}
					],
					"zombies": 4,
					"zone": "upstream_test_zone"
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

		if payload != "" {
			rw.WriteHeader(http.StatusOK)
			_, err := rw.Write([]byte(payload))

			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)

			return
		}

		rw.WriteHeader(http.StatusNotFound)
	}))
}

func CreateNginxPlusUpstreamServer(t *testing.T) client.UpstreamServer {
	t.Helper()

	maxConns := 10
	maxFails := 2
	weight := 0
	down := false
	backup := true

	return client.UpstreamServer{
		MaxConns:    &maxConns,
		MaxFails:    &maxFails,
		Backup:      &backup,
		Down:        &down,
		Weight:      &weight,
		Server:      "test_server",
		FailTimeout: "",
		SlowStart:   "",
		Route:       "",
		Service:     "",
		ID:          serverID,
		Drain:       false,
	}
}

func CreateNginxPlusStreamServer(t *testing.T) client.StreamUpstreamServer {
	t.Helper()

	maxConns := 10
	maxFails := 2
	weight := 0
	down := false
	backup := true

	return client.StreamUpstreamServer{
		MaxConns:    &maxConns,
		MaxFails:    &maxFails,
		Backup:      &backup,
		Down:        &down,
		Weight:      &weight,
		Server:      "test_server",
		FailTimeout: "",
		SlowStart:   "",
		Service:     "",
		ID:          serverID,
	}
}
