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
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/nginxinc/nginx-prometheus-exporter/client"
	log "github.com/sirupsen/logrus"
)

type NginxOSS struct {
	baseDimensions *metrics.CommonDim
	stubStatus     string
	*namedMetric
	// This is for keeping the previous stats.  Need to report the delta.
	prevStats *client.StubStats
	init      sync.Once
	logger    *MetricSourceLogger
}

func NewNginxOSS(baseDimensions *metrics.CommonDim, namespace, stubStatus string) *NginxOSS {
	return &NginxOSS{baseDimensions: baseDimensions, stubStatus: stubStatus, namedMetric: &namedMetric{namespace: namespace}, logger: NewMetricSourceLogger()}
}

func (c *NginxOSS) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	c.init.Do(func() {
		cl, err := client.NewNginxClient(&http.Client{}, c.stubStatus)
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to create oss metrics client, %v", err))
			c.prevStats = nil
			return
		}

		c.prevStats, err = cl.GetStubStats()
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to retrieve oss metrics, %v", err))
			c.prevStats = nil
			return
		}
	})

	cl, err := client.NewNginxClient(&http.Client{}, c.stubStatus)
	if err != nil {
		c.logger.Log(fmt.Sprintf("Failed to create oss metrics client, %v", err))
		SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
		return
	}

	stats, err := cl.GetStubStats()
	if err != nil {
		c.logger.Log(fmt.Sprintf("Failed to retrieve oss metrics, %v", err))
		SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
		return
	}

	if c.prevStats == nil {
		c.prevStats = stats
	}

	c.baseDimensions.NginxType = OSSNginxType
	c.baseDimensions.PublishedAPI = c.stubStatus
	c.group = "http"

	connAccepted := stats.Connections.Accepted - c.prevStats.Connections.Accepted
	if stats.Connections.Accepted < c.prevStats.Connections.Accepted {
		connAccepted = stats.Connections.Accepted
	}
	connHandled := stats.Connections.Handled - c.prevStats.Connections.Handled
	if stats.Connections.Handled < c.prevStats.Connections.Handled {
		connHandled = stats.Connections.Handled
	}
	connDropped := (stats.Connections.Accepted - c.prevStats.Connections.Accepted) - (stats.Connections.Handled - c.prevStats.Connections.Handled)
	if stats.Connections.Accepted < c.prevStats.Connections.Accepted || stats.Connections.Handled < c.prevStats.Connections.Handled {
		connDropped = stats.Connections.Accepted - stats.Connections.Handled
	}
	requestCount := stats.Requests - c.prevStats.Requests
	if stats.Requests < c.prevStats.Requests {
		requestCount = stats.Requests
	}

	simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
		"conn.active":     float64(stats.Connections.Active),
		"conn.accepted":   float64(connAccepted),
		"conn.handled":    float64(connHandled),
		"conn.current":    float64(stats.Connections.Active + stats.Connections.Waiting),
		"conn.idle":       float64(stats.Connections.Waiting),
		"conn.dropped":    float64(connDropped),
		"conn.reading":    float64(stats.Connections.Reading),
		"conn.writing":    float64(stats.Connections.Writing),
		"request.count":   float64(requestCount),
		"request.current": float64(stats.Connections.Reading) + float64(stats.Connections.Writing),
	})

	simpleMetrics = append(simpleMetrics, &proto.SimpleMetric{Name: "nginx.status", Value: 1.0})

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntity(c.baseDimensions.ToDimensions(), simpleMetrics):
	}

	c.prevStats = stats
}

func (c *NginxOSS) Stop() {
	log.Debugf("Stopping NginxOSS source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxOSS) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.baseDimensions = dimensions
	c.stubStatus = collectorConf.StubStatus
}
