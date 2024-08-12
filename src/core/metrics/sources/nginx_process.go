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

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	log "github.com/sirupsen/logrus"
)

type NginxProcess struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
	binary core.NginxBinary
}

// NewNginxProc collects metrics about nginx and nginx child processes
func NewNginxProcess(baseDimensions *metrics.CommonDim,
	namespace string,
	binary core.NginxBinary,
) *NginxProcess {
	return &NginxProcess{
		baseDimensions: baseDimensions,
		namedMetric:    &namedMetric{namespace: namespace},
		binary:         binary,
	}
}

// Get live NGINX Plus status
func (c *NginxProcess) getNginxCount() float64 {
	details := c.binary.GetNginxDetailsByID(c.baseDimensions.NginxId)
	if details != nil && details.NginxId != "" && c.baseDimensions.NginxId != "" && details.Plus.Enabled {
		return boolToFloat64(details.NginxId == c.baseDimensions.NginxId)
	}
	return 0.0
}

func (c *NginxProcess) Collect(ctx context.Context, _ *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	// defer wg.Done()

	l := &namedMetric{namespace: PlusNamespace, group: ""}
	countSimpleMetric := l.convertSamplesToSimpleMetrics(map[string]float64{
		"instance.count": c.getNginxCount(),
	})

	log.Debugf("instance metrics count %d", len(countSimpleMetric))

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntityWrapper(c.baseDimensions.ToDimensions(), countSimpleMetric, proto.MetricsReport_INSTANCE):
	}
}

func (c *NginxProcess) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.baseDimensions = dimensions
}

func (c *NginxProcess) Stop() {
	log.Debugf("Stopping NginxProcess source for nginx id: %v", c.baseDimensions.NginxId)
}
