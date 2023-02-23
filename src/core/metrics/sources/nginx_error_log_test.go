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

func TestNginxErrorLogUpdate(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetErrorLogs").Return(map[string]string{"/tmp/error.log": ""}).Once()
	binary.On("GetErrorLogs").Return(map[string]string{"/tmp/new_error.log": ""}).Once()

	collectionDuration := time.Millisecond * 300
	newCollectionDuration := time.Millisecond * 500
	nginxErrorLog := NewNginxErrorLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	assert.Equal(t, "", nginxErrorLog.baseDimensions.InstanceTags)
	assert.Equal(t, collectionDuration, nginxErrorLog.collectionInterval)
	_, ok := nginxErrorLog.logs["/tmp/error.log"]
	assert.True(t, ok)

	nginxErrorLog.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{
			CollectionInterval: newCollectionDuration,
		},
	)

	assert.Equal(t, "new-tag", nginxErrorLog.baseDimensions.InstanceTags)
	assert.Equal(t, newCollectionDuration, nginxErrorLog.collectionInterval)
	_, ok = nginxErrorLog.logs["/tmp/new_error.log"]
	assert.True(t, ok)
}

func TestNginxErrorLogStop(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetErrorLogs").Return(map[string]string{"/tmp/error.log": ""}).Once()

	collectionDuration := time.Millisecond * 300
	nginxErrorLog := NewNginxErrorLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)

	_, ok := nginxErrorLog.logs["/tmp/error.log"]
	assert.True(t, ok)

	nginxErrorLog.Stop()

	assert.Len(t, nginxErrorLog.logs, 0)
}

func TestErrorLogStats(t *testing.T) {
	tests := []struct {
		name          string
		logLines      []string
		expectedStats *proto.StatsEntity
	}{
		{
			"default_error_log_test",
			[]string{
				`2015/07/14 08:42:57 [error] 28386#28386: *38698 upstream timed out (110: Connection timed out) while reading response header from upstream, client: 127.0.0.1, server: localhost, request: "GET /1.0/ HTTP/1.0", upstream: "uwsgi://127.0.0.1:3131", host: "localhost:5000"`,
				`2015/07/15 05:56:33 [warn] 28386#28386: *94149 an upstream response is buffered to a temporary file /var/cache/nginx/proxy_temp/4/08/0000000084 while reading upstream, client: 85.141.232.177, server: *.compute.amazonaws.com, request: "POST /api/metrics/query/timeseries/ HTTP/1.1", upstream: "http://127.0.0.1:3000/api/metrics/query/timeseries/", host: "ec2-54-78-3-178.eu-west-1.compute.amazonaws.com:4000", referrer: "http://ec2-54-78-3-178.eu-west-1.compute.amazonaws.com:4000/"`,
				`2015/07/15 05:56:30 [info] 28386#28386: *94160 client 10.196.158.41 closed keepalive connection`,
				`2022/05/24 13:18:37 [error] 21314#21314: *91 connect() failed (111: Connection refused) while connecting to upstream, client: 127.0.0.1, server: , request: "GET /frontend1 HTTP/1.1", upstream: "http://127.0.0.1:9091/frontend1", host: "127.0.0.1:8081"`,
				`2022/05/24 13:18:37 [error] 21314#21314: client request body is buffered.`,
			},
			&proto.StatsEntity{
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "nginx.upstream.response.failed",
						Value: 1,
					},
					{
						Name:  "nginx.upstream.request.failed",
						Value: 1,
					},
					{
						Name:  "nginx.upstream.response.buffered",
						Value: 1,
					},
					{
						Name:  "nginx.http.request.buffered",
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
			errorLogFile, _ := os.CreateTemp(os.TempDir(), "error.log")

			nginxErrorLog := NewNginxErrorLog(&metrics.CommonDim{}, OSSNamespace, binary, OSSNginxType, collectionDuration)
			go nginxErrorLog.logStats(context.TODO(), errorLogFile.Name())

			time.Sleep(sleepDuration)

			for _, logLine := range test.logLines {
				_, err := errorLogFile.WriteString(logLine)
				require.NoError(t, err, "Error writing data to error log")
			}

			time.Sleep(collectionDuration)

			errorLogFile.Close()
			os.Remove(errorLogFile.Name())

			// Sort metrics before doing comparison
			sort.SliceStable(test.expectedStats.GetSimplemetrics(), func(i, j int) bool {
				return test.expectedStats.GetSimplemetrics()[i].Name < test.expectedStats.GetSimplemetrics()[j].Name
			})
			sort.SliceStable(nginxErrorLog.buf[0].Data.GetSimplemetrics(), func(i, j int) bool {
				return nginxErrorLog.buf[0].Data.GetSimplemetrics()[i].Name < nginxErrorLog.buf[0].Data.GetSimplemetrics()[j].Name
			})

			assert.Equal(tt, test.expectedStats.GetSimplemetrics(), nginxErrorLog.buf[0].Data.GetSimplemetrics())
		})
	}
}
