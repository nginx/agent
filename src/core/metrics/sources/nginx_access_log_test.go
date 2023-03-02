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
			`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time" "$gzip_ratio" "$server_protocol" "$upstream_connect_time" "$upstream_header_time" "$upstream_response_length" "$upstream_response_time" "$upstream_status" "$upstream_cache_status"`,
			[]string{
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"10\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"POST /nginx_status HTTP/1.1\" 201 98 \"-\" \"Go-http-client/1.1\" \"-\" \"250\" \"110\" \"0.300\" \"20\" \"HTTP/1.1\" \"350\" \"730\" \"28\" \"0.01\" \"201\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 400 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.03\" \"400\" \"MISS\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"DELETE /nginx_status HTTP/1.1\" 403 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"100\" \"500\" \"28\" \"0.00\" \"403\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"HEAD /nginx_status HTTP/1.1\" 404 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"505\" \"28\" \"0.00\" \"404\" \"MISS\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 499 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"350\" \"2000\" \"28\" \"0.00\" \"-\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"PUT /nginx_status HTTP/1.1\" 500 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"2350\" \"250\" \"28\" \"0.02\" \"500\" \"UPDATING\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.0\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.0\" \"350\" \"500\" \"28\" \"0.01\" \"502\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/2\" 503 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/2\" \"350\" \"500\" \"28\" \"0.00\" \"503\" \"HIT\" \n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/0.9\" 504 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/0.9\" \"350\" \"590\" \"28\" \"0.00\" \"502\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"OPTIONS /nginx_status HTTP/1.1\" 502 98 \"-\" \"Go-http-client/1.1\" \"-\" \"200\" \"100\" \"0.200\" \"-\" \"HTTP/1.1\" \"900\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\" \"350\" \"170\" \"28\" \"0.00\" \"200\" \"HIT\"\n",
				"127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"TRACE /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\" \"150\" \"105\" \"0.100\" \"-\" \"HTTP/1.1\" \"350\" \"500\" \"28\" \"0.00\" \"200\" \"HIT\"\n",
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
						Value: 514.2857142857143,
					},
					{
						Name:  "nginx.upstream.connect.time.count",
						Value: 14,
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
						Value: 588.9285714285714,
					},
					{
						Name:  "nginx.upstream.header.time.count",
						Value: 14,
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
						Name:  "nginx.upstream.response.time",
						Value: 0.005,
					},
					{
						Name:  "nginx.upstream.response.time.count",
						Value: 14,
					},
					{
						Name:  "nginx.upstream.response.time.max",
						Value: 0.03,
					},
					{
						Name:  "nginx.upstream.response.time.median",
						Value: 0,
					},
					{
						Name:  "nginx.upstream.response.time.pctl95",
						Value: 0.02,
					},
					{
						Name:  "nginx.upstream.response.length",
						Value: 28,
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
			sort.SliceStable(nginxAccessLog.buf[0].GetSimplemetrics(), func(i, j int) bool {
				return nginxAccessLog.buf[0].GetSimplemetrics()[i].Name < nginxAccessLog.buf[0].GetSimplemetrics()[j].Name
			})

			assert.Equal(tt, test.expectedStats.GetSimplemetrics(), nginxAccessLog.buf[0].GetSimplemetrics())
		})
	}
}
