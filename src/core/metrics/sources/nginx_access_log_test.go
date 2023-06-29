/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessLogUpdate(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetAccessLogs").Return(map[string]string{"/tmp/access.log": ""}).Once()
	binary.On("GetAccessLogs").Return(map[string]string{"/tmp/new_access.log": ""}).Once()

	collectionDuration := time.Millisecond * 300
	newCollectionDuration := time.Millisecond * 500
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	assert.Equal(t, "", nginxAccessLog.baseDimensions.InstanceTags)
	assert.Equal(t, collectionDuration, nginxAccessLog.collectionInterval)
	_, ok := nginxAccessLog.logs["/tmp/access.log"]
	assert.True(t, ok)

	nginxAccessLog.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{
			CollectionInterval: newCollectionDuration,
		},
	)

	assert.Equal(t, "new-tag", nginxAccessLog.baseDimensions.InstanceTags)
	assert.Equal(t, newCollectionDuration, nginxAccessLog.collectionInterval)
	_, ok = nginxAccessLog.logs["/tmp/new_access.log"]
	assert.True(t, ok)
}

func TestAccessLogStop(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetAccessLogs").Return(map[string]string{"/tmp/access.log": ""}).Once()

	collectionDuration := time.Millisecond * 300
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	_, ok := nginxAccessLog.logs["/tmp/access.log"]
	assert.True(t, ok)

	nginxAccessLog.Stop()

	assert.Len(t, nginxAccessLog.logs, 0)
}

func TestCalculateUpstreamNextCount(t *testing.T) {
	upstreamRequest := false

	tests := []struct {
		name                    string
		upstreamTimes           []string
		expectedUpstreamRequest bool
		expectedCount           float64
		upstreamCounters        map[string]float64
	}{
		{
			"singleUpstreamTimes",
			[]string{"0.01", "0.02", "0.00"},
			true,
			float64(0),
			map[string]float64{
				"upstream.next.count": 0,
			},
		},
		{
			"multipleUpstreamTimes",
			[]string{"0.01, 0.04, 0.03, 0.00", "0.02, 0.01, 0.03, 0.04", "0.00, 0.00, 0.08, 0.02"},
			true,
			float64(3),
			map[string]float64{
				"upstream.next.count": 0,
			},
		},
		{
			"noUpstreamTimes",
			[]string{"-", "-", "-"},
			false,
			float64(0),
			map[string]float64{
				"upstream.next.count": 0,
			},
		},
		{
			"emptyUpstreamTimes",
			[]string{"", "", ""},
			false,
			float64(0),
			map[string]float64{
				"upstream.next.count": 0,
			},
		},
		{
			"oneUpstreamTime",
			[]string{"-, -, -, 0.04", "-, -, -, 0.02", "-, -, -, 0.02"},
			true,
			float64(3),
			map[string]float64{
				"upstream.next.count": 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			upstreamRequest, test.upstreamCounters = calculateUpstreamNextCount(test.upstreamTimes, test.upstreamCounters)
			assert.Equal(t, test.expectedUpstreamRequest, upstreamRequest)
			assert.Equal(t, test.expectedCount, test.upstreamCounters["upstream.next.count"])
		})

	}

}

func TestParseAccessLogFloatTimes(t *testing.T) {
	tests := []struct {
		name            string
		metricName      string
		metric          string
		counter         []float64
		expectedCounter []float64
	}{
		{
			"validTime",
			"request_time",
			"0.00",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03, 0.00},
		},
		{
			"noTime",
			"request_time",
			"-",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
		{
			"emptyTime",
			"request_time",
			"",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
		{
			"invalidTime",
			"request_time",
			"test",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration := time.Millisecond * 300
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.counter = nginxAccessLog.parseAccessLogFloatTimes(test.metricName, test.metric, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)

		})
	}
}

func TestParseAccessLogUpstream(t *testing.T) {
	tests := []struct {
		name            string
		metricName      string
		metric          string
		counter         []float64
		expectedCounter []float64
	}{
		{
			"singleTime",
			"upstream_connect_time",
			"0.03",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03, 0.03},
		},
		{
			"multipleTimes",
			"upstream_connect_time",
			"0.03, 0.04, 0.06",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03, 0.03, 0.04, 0.06},
		},
		{
			"someEmptyTimes",
			"upstream_connect_time",
			"0.03, 0.04, -, 0.06, -",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03, 0.03, 0.04, 0.06},
		},
		{
			"emptyTime",
			"upstream_connect_time",
			"",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
		{
			"noTime",
			"upstream_connect_time",
			"-",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
		{
			"invalidTimes",
			"upstream_connect_time",
			"test",
			[]float64{0.02, 0.03},
			[]float64{0.02, 0.03},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration := time.Millisecond * 300
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.counter = nginxAccessLog.parseAccessLogUpstream(test.metricName, test.metric, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)

		})
	}
}

func TestParseAccessLogFloatCounters(t *testing.T) {
	tests := []struct {
		name            string
		metricName      string
		metric          string
		counter         map[string]float64
		expectedCounter map[string]float64
	}{
		{

			"validCount",
			"request.bytes_sent",
			"28",
			map[string]float64{
				"request.bytes_sent": 4,
			},
			map[string]float64{
				"request.bytes_sent": 32,
			},
		},
		{

			"noCount",
			"request.bytes_sent",
			"-",
			map[string]float64{
				"request.bytes_sent": 4,
			},
			map[string]float64{
				"request.bytes_sent": 4,
			},
		},
		{

			"emptyCount",
			"request.bytes_sent",
			"",
			map[string]float64{
				"request.bytes_sent": 4,
			},
			map[string]float64{
				"request.bytes_sent": 4,
			},
		},
		{

			"invalidCount",
			"request.bytes_sent",
			"test",
			map[string]float64{
				"request.bytes_sent": 4,
			},
			map[string]float64{
				"request.bytes_sent": 4,
			},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration := time.Millisecond * 300
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nginxAccessLog.parseAccessLogFloatCounters(test.metricName, test.metric, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)

		})
	}
}

func TestCalculateServerProtocol(t *testing.T) {
	tests := []struct {
		name            string
		protocol        string
		counters        map[string]float64
		expectedCounter map[string]float64
	}{
		{
			"validProtocol",
			"HTTP/1.1",
			map[string]float64{
				"v0_9": 0,
				"v1_0": 0,
				"v1_1": 2,
				"v2":   0,
			},
			map[string]float64{
				"v0_9": 0,
				"v1_0": 0,
				"v1_1": 3,
				"v2":   0,
			},
		},
		{
			"invalidProtocol",
			"",
			map[string]float64{
				"v0_9": 0,
				"v1_0": 0,
				"v1_1": 0,
				"v2":   0,
			},
			map[string]float64{
				"v0_9": 0,
				"v1_0": 0,
				"v1_1": 0,
				"v2":   0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			calculateServerProtocol(test.protocol, test.counters)
			assert.Equal(t, test.expectedCounter, test.counters)
		})

	}
}

func TestGetParsedRequest(t *testing.T) {
	tests := []struct {
		name             string
		request          string
		expectedMethod   string
		expectedURI      string
		expectedProtocol string
	}{
		{
			"validRequest",
			"GET /user/register?ahref<random>p' or '</random> HTTP/1.1",
			"GET",
			"/user/register?ahref<random>p' or '</random>",
			"HTTP/1.1",
		},
		{
			"emptyRequest",
			"",
			"",
			"",
			"",
		},
		{
			"invalidRequest",
			"GET /user/register?ahref<random>p' or '</random> HTTP1.1",
			"",
			"",
			"",
		},
		{
			"nospacesRequest",
			"GET/user/register?ahref<random>p'or'</random>HTTP1.1",
			"",
			"",
			"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			method, uri, protocol := getParsedRequest(test.request)
			assert.Equal(t, test.expectedMethod, method)
			assert.Equal(t, test.expectedURI, uri)
			assert.Equal(t, test.expectedProtocol, protocol)
		})

	}

}

func TestGetAverageMetricValue(t *testing.T) {
	tests := []struct {
		name            string
		metricValues    []float64
		expectedAverage float64
	}{
		{
			"validValues",
			[]float64{28, 28, 4, 28, 19, 0},
			17.833333333333332,
		},
		{
			"emptyValues",
			[]float64{},
			0.0,
		},
		{
			"zeroValues",
			[]float64{0, 0, 0, 0},
			0.0,
		},
		{
			"decimalValues",
			[]float64{0.02, 0.3, 0.06, 0.07},
			0.1125,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			average := getAverageMetricValue(test.metricValues)
			assert.Equal(t, test.expectedAverage, average)
		})

	}
}

func TestCalculateHTTPStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          string
		counter         map[string]float64
		expectedCounter map[string]float64
	}{
		{
			"validStatus",
			"403",
			map[string]float64{
				"status.403":        4,
				"status.404":        2,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        6,
				"status.5xx":        0,
				"request.malformed": 0,
			},
			map[string]float64{
				"status.403":        5,
				"status.404":        2,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        7,
				"status.5xx":        0,
				"request.malformed": 0,
			},
		},
		{
			"discardedStatus",
			"499",
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        0,
				"status.5xx":        0,
				"request.malformed": 0,
			},
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  1,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        1,
				"status.5xx":        0,
				"request.malformed": 0,
			},
		},
		{
			"malformedStatus",
			"400",
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        0,
				"status.5xx":        0,
				"request.malformed": 0,
			},
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        1,
				"status.5xx":        0,
				"request.malformed": 1,
			},
		},
		{
			"emptyStatus",
			"",
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        0,
				"status.5xx":        0,
				"request.malformed": 0,
			},
			map[string]float64{
				"status.403":        0,
				"status.404":        0,
				"status.500":        0,
				"status.502":        0,
				"status.503":        0,
				"status.504":        0,
				"status.discarded":  0,
				"status.1xx":        0,
				"status.2xx":        0,
				"status.3xx":        0,
				"status.4xx":        0,
				"status.5xx":        0,
				"request.malformed": 0,
			},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration := time.Millisecond * 300
	nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nginxAccessLog.calculateHttpStatus(test.status, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)

		})
	}
}

func TestCalculateUpstreamCacheStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          string
		counter         map[string]float64
		expectedCounter map[string]float64
	}{
		{
			"validStatus",
			"HIT",
			map[string]float64{
				"cache.bypass":      0,
				"cache.expired":     0,
				"cache.hit":         4,
				"cache.miss":        0,
				"cache.revalidated": 0,
				"cache.stale":       0,
				"cache.updating":    0,
			},
			map[string]float64{
				"cache.bypass":      0,
				"cache.expired":     0,
				"cache.hit":         5,
				"cache.miss":        0,
				"cache.revalidated": 0,
				"cache.stale":       0,
				"cache.updating":    0,
			},
		},
		{
			"invalidStatus",
			"",
			map[string]float64{
				"cache.bypass":      0,
				"cache.expired":     0,
				"cache.hit":         4,
				"cache.miss":        0,
				"cache.revalidated": 0,
				"cache.stale":       0,
				"cache.updating":    0,
			},
			map[string]float64{
				"cache.bypass":      0,
				"cache.expired":     0,
				"cache.hit":         4,
				"cache.miss":        0,
				"cache.revalidated": 0,
				"cache.stale":       0,
				"cache.updating":    0,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			calculateUpstreamCacheStatus(test.status, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)
		})

	}
}

func TestGetTimeMetricsMap(t *testing.T) {
	tests := []struct {
		name            string
		metricName      string
		metricTimes     []float64
		counter         map[string]float64
		expectedCounter map[string]float64
	}{
		{
			"validMetrics",
			"request.time",
			[]float64{0.02, 0.09, 0.3, 0.8, 0.05},
			map[string]float64{
				"request.time":        0,
				"request.time.count":  0,
				"request.time.max":    0,
				"request.time.median": 0,
				"request.time.pctl95": 0,
			},
			map[string]float64{
				"request.time":        0.252,
				"request.time.count":  5,
				"request.time.max":    0.8,
				"request.time.median": 0.09,
				"request.time.pctl95": 0.8,
			},
		},
		{
			"emptyMetrics",
			"request.time",
			[]float64{},
			map[string]float64{
				"request.time":        0,
				"request.time.count":  0,
				"request.time.max":    0,
				"request.time.median": 0,
				"request.time.pctl95": 0,
			},
			map[string]float64{
				"request.time":        0,
				"request.time.count":  0,
				"request.time.max":    0,
				"request.time.median": 0,
				"request.time.pctl95": 0,
			},
		},
		{
			"singleMetric",
			"request.time",
			[]float64{0.07},
			map[string]float64{
				"request.time":        0,
				"request.time.count":  0,
				"request.time.max":    0,
				"request.time.median": 0,
				"request.time.pctl95": 0,
			},
			map[string]float64{
				"request.time":        0.07,
				"request.time.count":  1,
				"request.time.max":    0.07,
				"request.time.median": 0.07,
				"request.time.pctl95": 0.07,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			calculateTimeMetricsMap(test.metricName, test.metricTimes, test.counter)
			assert.Equal(t, test.expectedCounter, test.counter)
		})

	}
}

func TestAccessLogStats(t *testing.T) {
	tests := []struct {
		name          string
		logFormat     string
		logLines      []string
		expectedStats *proto.StatsEntity
	}{
		{
			"default_access_log_test",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n",
				`127.0.0.1 - - [19/May/2022:09:30:39 +0000] "GET /user/register?ahref<Script>p' or 's' = 's</Script> HTTP/1.1" 200 98 "-" "-" "-"`,
			},
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 196,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 0,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 2,
					},
					{
						Name:  "nginx.http.v2",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.request.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.next.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.length",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.2xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.cache.bypass",
						Value: 0,
					},
					{
						Name:  "nginx.cache.expired",
						Value: 0,
					},
					{
						Name:  "nginx.cache.hit",
						Value: 0,
					},
					{
						Name:  "nginx.cache.miss",
						Value: 0,
					},
					{
						Name:  "nginx.cache.revalidated",
						Value: 0,
					},
					{
						Name:  "nginx.cache.stale",
						Value: 0,
					},
					{
						Name:  "nginx.cache.updating",
						Value: 0,
					},
				},
			},
		},
		{
			"invalid_access_log",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n",
				"127.0.0.1 - - [09/Feb/2023:10:30:32 +0000] \"\x16\x03\x01\x02\x00\x01\x00\x01\xFC\x03\x03\xC1\x9F\xFD\x873E\x83%\x89hh\x8F\xC7\xD6\x14\xC3\x01\x84\xB8\xF3\x00ZPt\xAF\xD2\xE8x\x05\x16\x8DU \xB9>@\x15\xDA5\xC7\xCC\xB7N-\x84\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n",
			},
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 196,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 0,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 1,
					},
					{
						Name:  "nginx.http.v2",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.next.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.request.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.length",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.2xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.cache.bypass",
						Value: 0,
					},
					{
						Name:  "nginx.cache.expired",
						Value: 0,
					},
					{
						Name:  "nginx.cache.hit",
						Value: 0,
					},
					{
						Name:  "nginx.cache.miss",
						Value: 0,
					},
					{
						Name:  "nginx.cache.revalidated",
						Value: 0,
					},
					{
						Name:  "nginx.cache.stale",
						Value: 0,
					},
					{
						Name:  "nginx.cache.updating",
						Value: 0,
					},
				},
			},
		},
		{
			"full_access_log_test",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time" "$gzip_ratio" "$server_protocol" "$upstream_connect_time" "$upstream_header_time" "$upstream_response_length" "$upstream_response_time" "$upstream_status" "$upstream_cache_status" "$bar"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"10\" \"HTTP/1.1\" \"350, 0.001, 0.02, -\" \"500, 0.02, -, 20\" \"28, 0, 0, 2\" \"0.00, 0.03, 0.04, -\" \"200\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"POST /nginx_status HTTP/1.1\" 201 98 \"-\" \"Go-http-client/1.1\" \"-\" \"250\" \"110\" \"0.300\" \"20\" \"HTTP/1.1\" \"350, 0.01\" \"730, 80\" \"28, 28\" \"0.01, 0.02\" \"201\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 400 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.03\" \"400\" \"MISS\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 403 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"100\" \"500\" \"28\" \"0.00\" \"403\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"HEAD /nginx_status HTTP/1.1\" 404 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"505\" \"28\" \"0.00\" \"404\" \"MISS\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 499 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"2000\" \"28\" \"0.00\" \"-\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 500 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"2350\" \"250\" \"28\" \"0.02\" \"500\" \"UPDATING\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.0\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.0\" \"350\" \"500\" \"28\" \"0.01\" \"502\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/2\" 503 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/2\" \"350\" \"500\" \"28\" \"0.00\" \"503\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/0.9\" 504 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/0.9\" \"350\" \"590\" \"28\" \"0.00\" \"502\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.1\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"900\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\" \"350\" \"170\" \"28\" \"0.00\" \"200\" \"HIT\" \"bar\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\" \"bar\"\n",
			},
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 15,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0.18571428571428572,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 14,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0.2,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0.3,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0.2,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 1372,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 2700,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 101.78571428571429,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 1,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 4,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 5,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 4,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 5,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 1,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 1,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 1,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 1,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 11,
					},
					{
						Name:  "nginx.http.v2",
						Value: 1,
					},
					{
						Name:  "nginx.upstream.connect.time",
						Value: 423.53123529411766,
					},
					{
						Name:  "nginx.upstream.connect.time.count",
						Value: 17,
					},
					{
						Name:  "nginx.upstream.connect.time.max",
						Value: 2350,
					},
					{
						Name:  "nginx.upstream.connect.time.median",
						Value: 350,
					},
					{
						Name:  "nginx.upstream.connect.time.pctl95",
						Value: 900,
					},
					{
						Name:  "nginx.upstream.header.time",
						Value: 490.88352941176475,
					},
					{
						Name:  "nginx.upstream.header.time.count",
						Value: 17,
					},
					{
						Name:  "nginx.upstream.header.time.max",
						Value: 2000,
					},
					{
						Name:  "nginx.upstream.header.time.median",
						Value: 500,
					},
					{
						Name:  "nginx.upstream.header.time.pctl95",
						Value: 730,
					},
					{
						Name:  "nginx.upstream.request.count",
						Value: 14,
					},
					{
						Name:  "nginx.upstream.next.count",
						Value: 4,
					},
					{
						Name:  "nginx.upstream.response.time",
						Value: 0.009411764705882354,
					},
					{
						Name:  "nginx.upstream.response.time.count",
						Value: 17,
					},
					{
						Name:  "nginx.upstream.response.time.max",
						Value: 0.04,
					},
					{
						Name:  "nginx.upstream.response.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.pctl95",
						Value: 0.03,
					},
					{
						Name:  "nginx.upstream.response.length",
						Value: 23.444444444444443,
					},
					{
						Name:  "nginx.upstream.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.2xx",
						Value: 6,
					},
					{
						Name:  "nginx.upstream.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.4xx",
						Value: 3,
					},
					{
						Name:  "nginx.upstream.status.5xx",
						Value: 4,
					},
					{
						Name:  "nginx.cache.bypass",
						Value: 0,
					},
					{
						Name:  "nginx.cache.expired",
						Value: 0,
					},
					{
						Name:  "nginx.cache.hit",
						Value: 11,
					},
					{
						Name:  "nginx.cache.miss",
						Value: 2,
					},
					{
						Name:  "nginx.cache.revalidated",
						Value: 0,
					},
					{
						Name:  "nginx.cache.stale",
						Value: 0,
					},
					{
						Name:  "nginx.cache.updating",
						Value: 1,
					},
				},
			},
		},
		{
			"custom_access_log_test",
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$upstream_addr" "$upstream_addr_id" "$http_user_agent" "$http_x_forwarded_for" "$http_x_amplify_id"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"address\" \"address-id\" \"Go-http-client/1.1\" \"-\" \"xyz\"\n",
				`127.0.0.1 - - [19/May/2022:09:30:39 +0000] "GET /user/register?ahref<Script>p' or 's' = 's</Script> HTTP/1.1" 200 98 "-" "address" "address-id" "-" "-" "xyz"`,
			},
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.http.gzip.ratio",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.body_bytes_sent",
						Value: 196,
					},
					{
						Name:  "nginx.http.request.bytes_sent",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.length",
						Value: 0,
					},
					{
						Name:  "nginx.http.request.malformed",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.post",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.get",
						Value: 2,
					},
					{
						Name:  "nginx.http.method.delete",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.put",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.head",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.options",
						Value: 0,
					},
					{
						Name:  "nginx.http.method.others",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.2xx",
						Value: 2,
					},
					{
						Name:  "nginx.http.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.403",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.404",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.500",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.502",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.503",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.504",
						Value: 0,
					},
					{
						Name:  "nginx.http.status.discarded",
						Value: 0,
					},
					{
						Name:  "nginx.http.v0_9",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_0",
						Value: 0,
					},
					{
						Name:  "nginx.http.v1_1",
						Value: 2,
					},
					{
						Name:  "nginx.http.v2",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.connect.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.header.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.request.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.next.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.length",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.count",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.max",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.pctl95",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.1xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.2xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.3xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.4xx",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.status.5xx",
						Value: 0,
					},
					{
						Name:  "nginx.cache.bypass",
						Value: 0,
					},
					{
						Name:  "nginx.cache.expired",
						Value: 0,
					},
					{
						Name:  "nginx.cache.hit",
						Value: 0,
					},
					{
						Name:  "nginx.cache.miss",
						Value: 0,
					},
					{
						Name:  "nginx.cache.revalidated",
						Value: 0,
					},
					{
						Name:  "nginx.cache.stale",
						Value: 0,
					},
					{
						Name:  "nginx.cache.updating",
						Value: 0,
					},
				},
			},
		},
	}

	binary := core.NewNginxBinary(tutils.NewMockEnvironment(), &config.Config{})
	collectionDuration := time.Millisecond * 300
	sleepDuration := time.Millisecond * 100

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			accessLogFile, _ := os.CreateTemp(os.TempDir(), "access.log")

			nginxAccessLog := NewNginxAccessLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)
			go nginxAccessLog.logStats(context.TODO(), accessLogFile.Name(), test.logFormat)

			time.Sleep(sleepDuration)

			for _, logLine := range test.logLines {
				_, err := accessLogFile.WriteString(logLine)
				require.NoError(t, err, "Error writing data to access log")
			}

			time.Sleep(collectionDuration)

			accessLogFile.Close()
			os.Remove(accessLogFile.Name())

			// Sort metrics before doing comparison
			sort.SliceStable(test.expectedStats.GetSimplemetrics(), func(i, j int) bool {
				return test.expectedStats.GetSimplemetrics()[i].Name < test.expectedStats.GetSimplemetrics()[j].Name
			})
			sort.SliceStable(nginxAccessLog.buf[0].Data.GetSimplemetrics(), func(i, j int) bool {
				return nginxAccessLog.buf[0].Data.GetSimplemetrics()[i].Name < nginxAccessLog.buf[0].Data.GetSimplemetrics()[j].Name
			})

			assert.Equal(tt, test.expectedStats.GetSimplemetrics(), nginxAccessLog.buf[0].Data.GetSimplemetrics())
		})
	}
}
