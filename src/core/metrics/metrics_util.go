/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"

	"github.com/gogo/protobuf/types"
)

type Collector interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
	UpdateConfig(config *config.Config)
}
type Source interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
}
type NginxSource interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
	Update(dimensions *CommonDim, collectorConf *NginxCollectorConfig)
	Stop()
}
type NginxCollectorConfig struct {
	NginxId            string
	StubStatus         string
	PlusAPI            string
	BinPath            string
	ConfPath           string
	CollectionInterval time.Duration
	AccessLogs         []string
	ErrorLogs          []string
	ClientVersion      int
}

func NewStatsEntity(dims []*proto.Dimension, samples []*proto.SimpleMetric) *proto.StatsEntity {
	return &proto.StatsEntity{
		Timestamp:     types.TimestampNow(),
		Dimensions:    dims,
		Simplemetrics: samples,
	}
}

func GetTimeMetrics(times []float64, metricType string) float64 {
	if len(times) == 0 {
		return 0
	}

	switch metricType {
	case "time":
		// Calculate average
		sum := 0.0
		for _, t := range times {
			sum += t
		}

		return (math.Round(sum*1000) / 1000) / float64(len(times))

	case "count":
		return float64(len(times))

	case "max":
		sort.Float64s(times)
		return times[len(times)-1]

	case "median":
		sort.Float64s(times)

		mNumber := len(times) / 2
		if len(times)%2 != 0 {
			return times[mNumber]
		} else {
			return (times[mNumber-1] + times[mNumber]) / 2
		}

	case "pctl95":
		sort.Float64s(times)

		index := int(math.RoundToEven(float64(0.95)*float64(len(times)))) - 1
		return times[index]
	}

	return 0
}

func GetCalculationMap() map[string]string {
	return map[string]string{
		"system.cpu.idle":                                    "avg",
		"system.cpu.iowait":                                  "avg",
		"system.cpu.stolen":                                  "avg",
		"system.cpu.system":                                  "avg",
		"system.cpu.user":                                    "avg",
		"system.disk.free":                                   "avg",
		"system.disk.in_use":                                 "avg",
		"system.disk.total":                                  "avg",
		"system.disk.used":                                   "avg",
		"system.io.iops_r":                                   "sum",
		"system.io.iops_w":                                   "sum",
		"system.io.kbs_r":                                    "sum",
		"system.io.kbs_w":                                    "sum",
		"system.io.wait_r":                                   "sum",
		"system.io.wait_w":                                   "sum",
		"system.mem.available":                               "avg",
		"system.mem.buffered":                                "avg",
		"system.mem.cached":                                  "avg",
		"system.mem.free":                                    "avg",
		"system.mem.pct_used":                                "avg",
		"system.mem.shared":                                  "avg",
		"system.mem.total":                                   "avg",
		"system.mem.used":                                    "avg",
		"system.mem.used.all":                                "avg",
		"system.load.1":                                      "avg",
		"system.load.15":                                     "avg",
		"system.load.5":                                      "avg",
		"system.swap.free":                                   "avg",
		"system.swap.pct_free":                               "avg",
		"system.swap.total":                                  "avg",
		"system.swap.used":                                   "avg",
		"system.net.bytes_rcvd":                              "sum",
		"system.net.bytes_sent":                              "sum",
		"system.net.drops_in.count":                          "sum",
		"system.net.drops_out.count":                         "sum",
		"system.net.listen_overflows":                        "sum",
		"system.net.packets_in.count":                        "sum",
		"system.net.packets_in.error":                        "sum",
		"system.net.packets_out.count":                       "sum",
		"system.net.packets_out.error":                       "sum",
		"nginx.status":                                       "boolean",
		"nginx.config.generation":                            "sum",
		"nginx.http.gzip.ratio":                              "avg",
		"nginx.http.status.1xx":                              "sum",
		"nginx.http.status.2xx":                              "sum",
		"nginx.http.status.3xx":                              "sum",
		"nginx.http.status.4xx":                              "sum",
		"nginx.http.status.5xx":                              "sum",
		"nginx.http.status.403":                              "sum",
		"nginx.http.status.404":                              "sum",
		"nginx.http.status.500":                              "sum",
		"nginx.http.status.502":                              "sum",
		"nginx.http.status.503":                              "sum",
		"nginx.http.status.504":                              "sum",
		"nginx.http.status.discarded":                        "sum",
		"nginx.http.method.delete":                           "sum",
		"nginx.http.method.get":                              "sum",
		"nginx.http.method.head":                             "sum",
		"nginx.http.method.options":                          "sum",
		"nginx.http.method.post":                             "sum",
		"nginx.http.method.put":                              "sum",
		"nginx.http.method.others":                           "sum",
		"nginx.http.request.bytes_sent":                      "sum",
		"nginx.http.request.body_bytes_sent":                 "sum",
		"nginx.http.request.length":                          "avg",
		"nginx.http.request.malformed":                       "sum",
		"nginx.http.request.time":                            "avg",
		"nginx.http.request.time.count":                      "sum",
		"nginx.http.request.time.max":                        "avg",
		"nginx.http.request.time.median":                     "avg",
		"nginx.http.request.time.pctl95":                     "avg",
		"nginx.http.request.count":                           "sum",
		"nginx.http.request.current":                         "avg",
		"nginx.http.request.buffered":                        "sum",
		"nginx.http.v0_9":                                    "sum",
		"nginx.http.v1_0":                                    "sum",
		"nginx.http.v1_1":                                    "sum",
		"nginx.http.v2":                                      "sum",
		"nginx.upstream.connect.time":                        "avg",
		"nginx.upstream.connect.time.count":                  "sum",
		"nginx.upstream.connect.time.max":                    "avg",
		"nginx.upstream.connect.time.median":                 "avg",
		"nginx.upstream.connect.time.pctl95":                 "avg",
		"nginx.upstream.header.time":                         "avg",
		"nginx.upstream.header.time.count":                   "sum",
		"nginx.upstream.header.time.max":                     "avg",
		"nginx.upstream.header.time.median":                  "avg",
		"nginx.upstream.header.time.pctl95":                  "avg",
		"nginx.upstream.response.length":                     "avg",
		"nginx.upstream.response.time":                       "avg",
		"nginx.upstream.response.time.count":                 "sum",
		"nginx.upstream.response.time.max":                   "avg",
		"nginx.upstream.response.time.median":                "avg",
		"nginx.upstream.response.time.pctl95":                "avg",
		"nginx.upstream.status.1xx":                          "sum",
		"nginx.upstream.status.2xx":                          "sum",
		"nginx.upstream.status.3xx":                          "sum",
		"nginx.upstream.status.4xx":                          "sum",
		"nginx.upstream.status.5xx":                          "sum",
		"nginx.http.conn.handled":                            "sum",
		"nginx.http.conn.reading":                            "avg",
		"nginx.http.conn.writing":                            "avg",
		"nginx.http.conn.accepted":                           "sum",
		"nginx.http.conn.active":                             "avg",
		"nginx.http.conn.current":                            "avg",
		"nginx.http.conn.dropped":                            "sum",
		"nginx.http.conn.idle":                               "avg",
		"nginx.upstream.response.buffered":                   "sum",
		"nginx.upstream.request.failed":                      "sum",
		"nginx.upstream.response.failed":                     "sum",
		"nginx.workers.count":                                "avg",
		"nginx.workers.rlimit_nofile":                        "avg",
		"nginx.workers.cpu.user":                             "sum",
		"nginx.workers.cpu.system":                           "sum",
		"nginx.workers.cpu.total":                            "sum",
		"nginx.workers.fds_count":                            "avg",
		"nginx.workers.mem.vms":                              "sum",
		"nginx.workers.mem.rss":                              "sum",
		"nginx.workers.mem.rss_pct":                          "avg",
		"nginx.workers.io.kbs_r":                             "sum",
		"nginx.workers.io.kbs_w":                             "sum",
		"plus.http.limit_conns.passed":                       "sum",
		"plus.http.limit_conns.rejected":                     "sum",
		"plus.http.limit_conns.rejected_dry_run":             "sum",
		"plus.http.limit_reqs.passed":                        "sum",
		"plus.http.limit_reqs.delayed":                       "sum",
		"plus.http.limit_reqs.rejected":                      "sum",
		"plus.http.limit_reqs.delayed_dry_run":               "sum",
		"plus.http.limit_reqs.rejected_dry_run":              "sum",
		"plus.cache.bypass.responses":                        "sum",
		"plus.cache.bypass.bytes":                            "sum",
		"plus.cache.expired.responses":                       "sum",
		"plus.cache.expired.bytes":                           "sum",
		"plus.cache.hit.responses":                           "sum",
		"plus.cache.hit.bytes":                               "sum",
		"plus.cache.miss.responses":                          "sum",
		"plus.cache.miss.bytes":                              "sum",
		"plus.cache.revalidated.responses":                   "sum",
		"plus.cache.revalidated.bytes":                       "sum",
		"plus.cache.size":                                    "avg",
		"plus.cache.max_size":                                "avg",
		"plus.cache.stale.responses":                         "sum",
		"plus.cache.stale.bytes":                             "sum",
		"plus.cache.updating.responses":                      "sum",
		"plus.cache.updating.bytes":                          "sum",
		"plus.http.request.bytes_rcvd":                       "sum",
		"plus.http.request.bytes_sent":                       "sum",
		"plus.http.request.count":                            "sum",
		"plus.http.response.count":                           "sum",
		"plus.ssl.failed":                                    "sum",
		"plus.ssl.handshakes":                                "sum",
		"plus.ssl.reuses":                                    "sum",
		"plus.http.status.1xx":                               "sum",
		"plus.http.status.2xx":                               "sum",
		"plus.http.status.3xx":                               "sum",
		"plus.http.status.4xx":                               "sum",
		"plus.http.status.5xx":                               "sum",
		"plus.http.status.discarded":                         "sum",
		"plus.http.status.processing":                        "avg",
		"plus.stream.bytes_rcvd":                             "sum",
		"plus.stream.bytes_sent":                             "sum",
		"plus.stream.connections":                            "sum",
		"plus.stream.processing":                             "avg",
		"plus.stream.discarded":                              "sum",
		"plus.stream.status.2xx":                             "sum",
		"plus.stream.status.4xx":                             "sum",
		"plus.stream.status.5xx":                             "sum",
		"plus.stream.status.total":                           "sum",
		"plus.http.upstream.zombies":                         "avg",
		"plus.http.upstream.keepalives":                      "avg",
		"plus.http.upstream.queue.maxsize":                   "avg",
		"plus.http.upstream.queue.overflows":                 "sum",
		"plus.http.upstream.queue.size":                      "avg",
		"plus.http.upstream.peers.conn.active":               "avg",
		"plus.http.upstream.peers.header_time":               "avg",
		"plus.http.upstream.peers.header_time.count":         "sum",
		"plus.http.upstream.peers.header_time.max":           "avg",
		"plus.http.upstream.peers.header_time.median":        "avg",
		"plus.http.upstream.peers.header_time.pctl95":        "avg",
		"plus.http.upstream.peers.response.time":             "avg",
		"plus.http.upstream.peers.response.time.count":       "sum",
		"plus.http.upstream.peers.response.time.max":         "avg",
		"plus.http.upstream.peers.response.time.median":      "avg",
		"plus.http.upstream.peers.response.time.pctl95":      "avg",
		"plus.http.upstream.peers.request.count":             "sum",
		"plus.http.upstream.peers.response.count":            "sum",
		"plus.http.upstream.peers.status.1xx":                "sum",
		"plus.http.upstream.peers.status.2xx":                "sum",
		"plus.http.upstream.peers.status.3xx":                "sum",
		"plus.http.upstream.peers.status.4xx":                "sum",
		"plus.http.upstream.peers.status.5xx":                "sum",
		"plus.http.upstream.peers.bytes_sent":                "sum",
		"plus.http.upstream.peers.bytes_rcvd":                "sum",
		"plus.http.upstream.peers.fails":                     "sum",
		"plus.http.upstream.peers.unavail":                   "sum",
		"plus.http.upstream.peers.health_checks.fails":       "sum",
		"plus.http.upstream.peers.health_checks.unhealthy":   "sum",
		"plus.http.upstream.peers.health_checks.checks":      "sum",
		"plus.http.upstream.peers.state.up":                  "avg",
		"plus.http.upstream.peers.state.draining":            "avg",
		"plus.http.upstream.peers.state.down":                "avg",
		"plus.http.upstream.peers.state.unavail":             "avg",
		"plus.http.upstream.peers.state.checking":            "avg",
		"plus.http.upstream.peers.state.unhealthy":           "avg",
		"plus.http.upstream.peers.total.up":                  "avg",
		"plus.http.upstream.peers.total.draining":            "avg",
		"plus.http.upstream.peers.total.down":                "avg",
		"plus.http.upstream.peers.total.unavail":             "avg",
		"plus.http.upstream.peers.total.checking":            "avg",
		"plus.http.upstream.peers.total.unhealthy":           "avg",
		"plus.stream.upstream.zombies":                       "avg",
		"plus.stream.upstream.peers.conn.active":             "avg",
		"plus.stream.upstream.peers.conn.count":              "sum",
		"plus.stream.upstream.peers.connect_time":            "avg",
		"plus.stream.upstream.peers.connect_time.count":      "sum",
		"plus.stream.upstream.peers.connect_time.max":        "avg",
		"plus.stream.upstream.peers.connect_time.median":     "avg",
		"plus.stream.upstream.peers.connect_time.pctl95":     "avg",
		"plus.stream.upstream.peers.ttfb":                    "avg",
		"plus.stream.upstream.peers.response.time":           "avg",
		"plus.stream.upstream.peers.response.time.count":     "sum",
		"plus.stream.upstream.peers.response.time.max":       "avg",
		"plus.stream.upstream.peers.response.time.median":    "avg",
		"plus.stream.upstream.peers.response.time.pctl95":    "avg",
		"plus.stream.upstream.peers.bytes_sent":              "sum",
		"plus.stream.upstream.peers.bytes_rcvd":              "sum",
		"plus.stream.upstream.peers.fails":                   "sum",
		"plus.stream.upstream.peers.unavail":                 "sum",
		"plus.stream.upstream.peers.health_checks.fails":     "sum",
		"plus.stream.upstream.peers.health_checks.unhealthy": "sum",
		"plus.stream.upstream.peers.health_checks.checks":    "sum",
		"plus.stream.upstream.peers.state.up":                "avg",
		"plus.stream.upstream.peers.state.draining":          "avg",
		"plus.stream.upstream.peers.state.down":              "avg",
		"plus.stream.upstream.peers.state.unavail":           "avg",
		"plus.stream.upstream.peers.state.checking":          "avg",
		"plus.stream.upstream.peers.state.unhealthy":         "avg",
		"plus.stream.upstream.peers.total.up":                "avg",
		"plus.stream.upstream.peers.total.draining":          "avg",
		"plus.stream.upstream.peers.total.down":              "avg",
		"plus.stream.upstream.peers.total.unavail":           "avg",
		"plus.stream.upstream.peers.total.checking":          "avg",
		"plus.stream.upstream.peers.total.unhealthy":         "avg",
		"plus.slab.pages.used":                               "avg",
		"plus.slab.pages.free":                               "avg",
		"plus.slab.pages.total":                              "avg",
		"plus.slab.pages.pct_used":                           "avg",
		"plus.instance.count":                                "avg",
		"container.cpu.cores":                                "avg",
		"container.cpu.period":                               "avg",
		"container.cpu.quota":                                "avg",
		"container.cpu.shares":                               "avg",
		"container.cpu.set.cores":                            "avg",
		"container.cpu.throttling.time":                      "avg",
		"container.cpu.throttling.throttled":                 "avg",
		"container.cpu.throttling.periods":                   "avg",
		"container.cpu.throttling.percent":                   "avg",
		"container.mem.oom":                                  "avg",
		"container.mem.oom.kill":                             "avg",
	}
}
