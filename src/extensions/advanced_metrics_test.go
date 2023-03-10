/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestAppCentricMetric_toMetricReport(t *testing.T) {
	commonDimensions := []*proto.Dimension{
		{
			Name:  "commonDimension",
			Value: "comDimValue",
		},
	}
	metrics := []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{
					Name:  "dim1",
					Value: "dim1Value",
				},
				{
					Name:  "dim2",
					Value: "dim2Value",
				},
				{
					Name:  familyDimension,
					Value: streamMetricFamilyDimensionValue,
				},
			},
			Metrics: []publisher.Metric{
				{
					Name: hitcountMetric,
					Values: publisher.MetricValues{
						Count: 1,
						Last:  0,
						Min:   0,
						Max:   0,
						Sum:   0,
					},
				},
				{
					Name: bytesRcvdMetric,
					Values: publisher.MetricValues{
						Count: 1,
						Last:  0,
						Min:   0,
						Max:   0,
						Sum:   200,
					},
				},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{
					Name:  "dim3",
					Value: "dim3Value",
				},
				{
					Name:  "dim2",
					Value: "dim2Value2",
				},
				{
					Name:  familyDimension,
					Value: "web",
				},
			},
			Metrics: []publisher.Metric{
				{
					Name: hitcountMetric,
					Values: publisher.MetricValues{
						Count: 11,
						Last:  0,
						Min:   0,
						Max:   0,
						Sum:   0,
					},
				},
				{
					Name: clientNetworkLatencyMetric,
					Values: publisher.MetricValues{
						Count: 1,
						Last:  0,
						Min:   0,
						Max:   100,
						Sum:   0,
					},
				},
				{
					Name: bytesRcvdMetric,
					Values: publisher.MetricValues{
						Count: 1,
						Last:  0,
						Min:   0,
						Max:   0,
						Sum:   200,
					},
				},
			},
		},
	}

	now := types.TimestampNow()
	report := toMetricReport(metrics, now, commonDimensions)

	assert.Equal(t, &proto.MetricsReport{
		Meta: &proto.Metadata{Timestamp: now},
		Type: proto.MetricsReport_INSTANCE,
		Data: []*proto.StatsEntity{
			{
				Timestamp: now,
				Dimensions: []*proto.Dimension{
					{
						Name:  "commonDimension",
						Value: "comDimValue",
					},
					{
						Name:  "dim1",
						Value: "dim1Value",
					},
					{
						Name:  "dim2",
						Value: "dim2Value",
					},
					{
						Name:  familyDimension,
						Value: streamMetricFamilyDimensionValue,
					},
				},
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "stream.connections",
						Value: 1,
					},
					{
						Name:  "stream." + bytesRcvdMetric,
						Value: 200,
					},
				},
			},
			{
				Timestamp: now,
				Dimensions: []*proto.Dimension{
					{
						Name:  "commonDimension",
						Value: "comDimValue",
					},
					{
						Name:  "dim3",
						Value: "dim3Value",
					},
					{
						Name:  "dim2",
						Value: "dim2Value2",
					},
					{
						Name:  familyDimension,
						Value: "web",
					},
				},
				Simplemetrics: []*proto.SimpleMetric{
					{
						Name:  "http.request.count",
						Value: 11,
					},
					{
						Name:  clientNetworkLatencyMetric + ".max",
						Value: 100,
					},
					{
						Name:  "http.request." + bytesRcvdMetric,
						Value: 200,
					},
				},
			},
		},
	}, report)
}

func TestAppCentricMetricClose(t *testing.T) {
	env := tutils.GetMockEnv()
	pluginUnderTest := NewAdvancedMetrics(env, &config.Config{}, nil)

	ctx, cancelCTX := context.WithCancel(context.Background())
	defer cancelCTX()

	messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{}, []core.ExtensionPlugin{pluginUnderTest})

	pluginUnderTest.Init(messagePipe)
	pluginUnderTest.Close()

	env.AssertExpectations(t)
}

func TestAppCentricMetricSubscriptions(t *testing.T) {
	pluginUnderTest := NewAdvancedMetrics(tutils.GetMockEnv(), &config.Config{}, nil)
	assert.Equal(t, []string{}, pluginUnderTest.Subscriptions())
}
