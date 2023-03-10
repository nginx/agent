/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/stretchr/testify/assert"
)

func TestNginxOSSUpdate(t *testing.T) {
	nginxOSS := NewNginxOSS(&metrics.CommonDim{}, OSSNamespace, "http://localhost:8080/api")

	assert.Equal(t, "", nginxOSS.baseDimensions.InstanceTags)

	nginxOSS.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{},
	)

	assert.Equal(t, "new-tag", nginxOSS.baseDimensions.InstanceTags)
}

func TestNginxOSS_Collect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/basic_status" {
			data := []byte(`Active connections: 1
server accepts handled requests
9 9 471
Reading: 0 Writing: 1 Waiting: 0
`)

			rw.Header().Add("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			rw.Header().Add("Content-Security-Policy", "default-src 'self'")
			_, err := rw.Write(data)
			if err != nil {
				t.Logf("Error writing test data")
			}
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("Internal Server Error")
			_, err := rw.Write(data)
			if err != nil {
				t.Logf("Error writing test data")
			}
		}
	}))
	defer server.Close()

	tests := []struct {
		name            string
		namedMetric     *namedMetric
		stubAPI         string
		m               chan *proto.StatsEntity
		expectedMetrics map[string]float64
	}{
		{
			"valid stub API",
			&namedMetric{namespace: "nginx", group: "http"},
			server.URL + "/basic_status",
			make(chan *proto.StatsEntity, 1),
			map[string]float64{
				"nginx.status":               float64(1),
				"nginx.http.conn.active":     float64(1),
				"nginx.http.conn.accepted":   float64(0),
				"nginx.http.conn.handled":    float64(0),
				"nginx.http.conn.reading":    float64(0),
				"nginx.http.conn.writing":    float64(1),
				"nginx.http.request.count":   float64(0),
				"nginx.http.request.current": float64(1),
				"nginx.http.conn.dropped":    float64(0),
				"nginx.http.conn.idle":       float64(0),
				"nginx.http.conn.current":    float64(1),
			},
		}, {
			"unknown stub API",
			&namedMetric{namespace: "nginx", group: "http"},
			server.URL + "/unknown",
			make(chan *proto.StatsEntity, 1),
			map[string]float64{
				"nginx.status": float64(0),
			},
		},
	}

	hostInfo := &proto.HostInfo{
		Hostname: "MyServer",
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			c := &NginxOSS{
				baseDimensions: metrics.NewCommonDim(hostInfo, &config.Config{}, ""),
				stubStatus:     test.stubAPI,
				namedMetric:    test.namedMetric,
				logger:         NewMetricSourceLogger(),
			}
			ctx := context.TODO()
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go c.Collect(ctx, wg, test.m)
			wg.Wait()
			statEntity := <-test.m
			assert.Len(tt, statEntity.Simplemetrics, len(test.expectedMetrics))
			for _, metric := range statEntity.Simplemetrics {
				assert.Contains(t, test.expectedMetrics, metric.Name)
				assert.Equal(t, test.expectedMetrics[metric.Name], metric.Value)
			}
		})
	}
}
