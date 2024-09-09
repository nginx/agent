/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"

	"github.com/nginx/agent/v2/src/core/metrics"
	log "github.com/sirupsen/logrus"
)

type NginxStatic struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
}

// This is to output a static "nginx.status = 0" metric if NGINX is not running or detected or NGINX Plus API or
// stub_status API is not setup properly.
func NewNginxStatic(baseDimensions *metrics.CommonDim, namespace string) *NginxStatic {
	return &NginxStatic{baseDimensions: baseDimensions, namedMetric: &namedMetric{namespace: namespace}}
}

func (c *NginxStatic) Collect(ctx context.Context, m chan<- *metrics.StatsEntityWrapper) {
	SendNginxDownStatus(ctx, c.baseDimensions.ToDimensions(), m)
}

func (c *NginxStatic) Stop() {
	log.Debugf("Stopping NginxStatic source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxStatic) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.baseDimensions = dimensions
}
