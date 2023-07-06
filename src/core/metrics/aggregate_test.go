/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"encoding/json"
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestSaveCollections(t *testing.T) {
	reports := []*proto.MetricsReport{
		{
			Meta: &proto.Metadata{},
			Type: proto.MetricsReport_SYSTEM,
			Data: []*proto.StatsEntity{
				{
					Simplemetrics: []*proto.SimpleMetric{
						{
							Name:  "system.mem.used",
							Value: 5,
						},
						{
							Name:  "system.io.kbs_w",
							Value: 5.3,
						},
						{
							Name:  "system.cpu.system",
							Value: 2.4,
						},
						{
							Name:  "system.cpu.user",
							Value: 6.8,
						},
					},
					Dimensions: []*proto.Dimension{
						{
							Name:  "hostname",
							Value: "test-host",
						},
					},
				},
			},
		},
		{
			Meta: &proto.Metadata{},
			Type: proto.MetricsReport_SYSTEM,
			Data: []*proto.StatsEntity{
				{
					Simplemetrics: []*proto.SimpleMetric{
						{
							Name:  "system.mem.used",
							Value: 6,
						},
						{
							Name:  "system.io.kbs_w",
							Value: 7.3,
						},
						{
							Name:  "system.cpu.system",
							Value: 8.3,
						},
						{
							Name:  "system.cpu.user",
							Value: 3.8,
						},
					},
					Dimensions: []*proto.Dimension{
						{
							Name:  "hostname",
							Value: "test-host2",
						},
					},
				},
			},
		},
	}

	metricsCollections := Collections{
		Count: 2,
		Data:  make(map[string]PerDimension),
	}
	dimension1 := []*proto.Dimension{
		{
			Name:  "hostname",
			Value: "test-host",
		},
	}
	var dimensionsChecksum string
	data, err := json.Marshal(dimension1)
	if err == nil {
		dimensionsChecksum = checksum.HexChecksum(data)
	} else {
		dimensionsChecksum = checksum.HexChecksum([]byte(fmt.Sprintf("%v", dimension1)))
	}
	metricsCollections.Data[dimensionsChecksum] = PerDimension{
		Dimensions:    dimension1,
		RunningSumMap: make(map[string]float64),
	}
	metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.mem.used"] = 6.2
	metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.io.kbs_w"] = 3.4
	metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.io.kbs_r"] = 2.3
	metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.cpu.system"] = 6.2

	metricsCollections = SaveCollections(metricsCollections, reports...)
	log.Info(metricsCollections)

	assert.NotNil(t, metricsCollections)
	assert.Equal(t, 11.2, metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.mem.used"])
	assert.Equal(t, 8.7, metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.io.kbs_w"])
	assert.Equal(t, 2.3, metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.io.kbs_r"])
	assert.Equal(t, 8.6, metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.cpu.system"])
	assert.Equal(t, 6.8, metricsCollections.Data[dimensionsChecksum].RunningSumMap["system.cpu.user"])
}

func TestGenerateAggregationReport(t *testing.T) {
	metricsCollections := Collections{
		Count: 2,
		Data:  make(map[string]PerDimension),
	}
	dimension1 := []*proto.Dimension{
		{
			Name:  "hostname",
			Value: "test-host",
		},
	}
	csum := checksum.HexChecksum([]byte(fmt.Sprintf("%v", dimension1)))
	metricsCollections.Data[csum] = PerDimension{
		Dimensions:    dimension1,
		RunningSumMap: make(map[string]float64),
	}
	metricsCollections.Data[csum].RunningSumMap["system.mem.used"] = 100.2
	metricsCollections.Data[csum].RunningSumMap["system.io.kbs_w"] = 600
	metricsCollections.Data[csum].RunningSumMap["system.io.kbs_r"] = 6000
	metricsCollections.Data[csum].RunningSumMap["system.cpu.system"] = 200.2
	metricsCollections.Data[csum].RunningSumMap["system.undefined_method"] = 1000

	results := GenerateMetrics(metricsCollections)
	log.Info(results)

	assert.NotEmpty(t, results)
	for _, stats := range results {
		simplemetrics := stats.GetSimplemetrics()
		for _, v := range simplemetrics {
			switch {
			case v.Name == "system.mem.used":
				assert.Equal(t, float64(50.1), v.Value)
			case v.Name == "system.io.kbs_w":
				assert.Equal(t, float64(600), v.Value)
			case v.Name == "system.io.kbs_r":
				assert.Equal(t, float64(6000), v.Value)
			case v.Name == "system.cpu.system":
				assert.Equal(t, float64(100.1), v.Value)
			case v.Name == "system.undefined_method":
				assert.Equal(t, float64(1000), v.Value)
			}
		}

	}
}

func TestAvg(t *testing.T) {
	result := avg(float64(2.12), 2)
	assert.Equal(t, float64(1.06), result)

	result = avg(float64(2.12), 0)
	assert.Equal(t, float64(2.12), result)
}

func TestSum(t *testing.T) {
	result := sum(float64(2.12), 2)
	assert.Equal(t, float64(2.12), result)
}

func TestBoolean(t *testing.T) {
	result := boolean(float64(2.12), 2)
	assert.Equal(t, 1.0, result)

	result = boolean(float64(0.2), 2)
	assert.Equal(t, 0.0, result)

	result = boolean(float64(2.12), 0)
	assert.Equal(t, 2.12, result)
}
