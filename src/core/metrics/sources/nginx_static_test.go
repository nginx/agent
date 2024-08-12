/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"

	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
)

func TestNginxStaticUpdate(t *testing.T) {
	nginxStatic := NewNginxStatic(&metrics.CommonDim{}, "test")

	assert.Equal(t, "", nginxStatic.baseDimensions.InstanceTags)

	nginxStatic.Update(
		&metrics.CommonDim{
			InstanceTags: "new-tag",
		},
		&metrics.NginxCollectorConfig{},
	)

	assert.Equal(t, "new-tag", nginxStatic.baseDimensions.InstanceTags)
}

func TestNginxStatic_Collect(t *testing.T) {
	expectedMetrics := map[string]bool{
		"nginx.status": true,
	}

	expectedMetricsValues := map[string]float64{
		"nginx.status": float64(0),
	}

	tests := []struct {
		name        string
		namedMetric *namedMetric
		m           chan *metrics.StatsEntityWrapper
	}{
		{
			"nginx static test",
			&namedMetric{namespace: "nginx"},
			make(chan *metrics.StatsEntityWrapper, 1),
		},
	}

	hostInfo := &proto.HostInfo{
		Hostname: "MyServer",
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			c := &NginxStatic{
				baseDimensions: metrics.NewCommonDim(hostInfo, &config.Config{}, ""),
				namedMetric:    test.namedMetric,
			}
			ctx := context.TODO()
			// wg := &sync.WaitGroup{}
			// wg.Add(1)
			go c.Collect(ctx, nil, test.m)
			// wg.Wait()
			statEntity := <-test.m
			assert.Len(tt, statEntity.Data.Simplemetrics, len(expectedMetrics))
			for _, metric := range statEntity.Data.Simplemetrics {
				assert.Contains(t, expectedMetrics, metric.Name)
				assert.Equal(t, expectedMetricsValues[metric.Name], metric.Value)
			}
		})
	}
}
