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
	"github.com/stretchr/testify/require"
)

var reports = []*proto.MetricsReport{
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

func TestSaveCollections(t *testing.T) {
	tests := []struct {
		name     string
		reports  []*proto.MetricsReport
		expected map[string]float64
	}{
		{
			name:    "save collection test",
			reports: reports,
			expected: map[string]float64{
				"system.mem.used":   5,
				"system.io.kbs_w":   5.3,
				"system.io.kbs_r":   0,
				"system.cpu.system": 2.4,
				"system.cpu.user":   6.8,
			},
		},
		{
			name:    "save collection test with duplicates",
			reports: append(reports, &proto.MetricsReport{
				Meta: &proto.Metadata{},
				Type: proto.MetricsReport_SYSTEM,
				Data: []*proto.StatsEntity{
					{
						Simplemetrics: []*proto.SimpleMetric{
							{
								Name:  "system.mem.used",
								Value: 7,
							},
							{
								Name:  "system.io.kbs_w",
								Value: 4.3,
							},
							{
								Name:  "system.cpu.system",
								Value: 2.3,
							},
							{
								Name:  "system.cpu.user",
								Value: 1.8,
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
			}), 
			expected: map[string]float64{
				"system.mem.used":   12,
				"system.io.kbs_w":   9.6,
				"system.io.kbs_r":   0,
				"system.cpu.system": 4.699999999999999,
				"system.cpu.user":   8.6,
			},
		},
	}

	for _, test := range tests {

		metricsCollections := Collections{
			Count: len(test.reports),
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

		metricsCollections = SaveCollections(metricsCollections, test.reports...)
		log.Info(metricsCollections)

		assert.NotNil(t, metricsCollections)
		// assert.Len(t, len(metricsCollections) 3

		for key, value := range test.expected {
			assert.Equal(t, value, metricsCollections.Data[dimensionsChecksum].RunningSumMap[key])
		}
	}
}

func TestGenerateMetrics(t *testing.T) {
	// Prepare a collection with some data
	metricsCollections := Collections{
		Count: 2,
		Data: map[string]PerDimension{
			"checksum1": {
				Dimensions: []*proto.Dimension{
					{Name: "name1", Value: "value1"},
				},
				RunningSumMap: map[string]float64{
					"system.mem.used": 20.0,
				},
			},
		},
	}

	// Generate the metrics
	metrics := GenerateMetrics(metricsCollections)

	// Check the generated metrics
	require.Len(t, metrics, 1)

	// Check the simple metrics
	simpleMetrics := metrics[0].GetSimplemetrics()
	require.Len(t, simpleMetrics, 1)
	assert.Equal(t, "system.mem.used", simpleMetrics[0].Name)
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

func TestGetAggregatedSimpleMetric(t *testing.T) {
	internalMap := map[string]float64{
		"system.mem.used":   10.0,
		"system.cpu.system": 20.0,
	}

	simpleMetrics := getAggregatedSimpleMetric(1, internalMap)

	require.Len(t, simpleMetrics, 2)
	assert.Equal(t, "system.mem.used", simpleMetrics[0].Name)
	assert.Equal(t, 10.0, simpleMetrics[0].GetValue())
	assert.Equal(t, "system.cpu.system", simpleMetrics[1].Name)
	assert.Equal(t, 20.0, simpleMetrics[1].GetValue())
}

func TestAvg(t *testing.T) {
	result := avg(2.12, 2)
	assert.Equal(t, 1.06, result)

	result = avg(2.12, 0)
	assert.Equal(t, 2.12, result)
}

func TestSum(t *testing.T) {
	result := sum(2.12, 2)
	assert.Equal(t, 2.12, result)
}

func TestBoolean(t *testing.T) {
	result := boolean(2.12, 2)
	assert.Equal(t, 1.0, result)

	result = boolean(0.2, 2)
	assert.Equal(t, 0.0, result)

	result = boolean(2.12, 0)
	assert.Equal(t, 2.12, result)
}
