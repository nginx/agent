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
	"regexp"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/proto"
)

var (
	failsRegex = regexp.MustCompile(`slab.slots.*.fails`)
	freeRegex  = regexp.MustCompile(`slab.slots.*.free`)
	reqsRegex  = regexp.MustCompile(`slab.slots.*.reqs`)
	usedRegex  = regexp.MustCompile(`slab.slots.*.used`)
)

type PerDimension struct {
	Dimensions    []*proto.Dimension
	RunningSumMap map[string]float64
}
type MetricsHandler func(float64, int) float64

type Collections struct {
	Count        int // this is the number of collections run.  Will use this to calculate the average.
	MetricsCount map[string]PerDimension
	Data         map[string]PerDimension
}

func dimChecksum(stats *proto.StatsEntity) string {
	dims := stats.GetDimensions()
	data, err := json.Marshal(dims)
	if err == nil {
		return checksum.HexChecksum(data)
	}

	return checksum.HexChecksum([]byte(fmt.Sprintf("%#v", dims)))
}

// SaveCollections loops through one or more reports and get all the raw metrics for the Collections
// Note this function operate on the Collections struct data directly.
func SaveCollections(metricsCollections Collections, reports ...*proto.MetricsReport) Collections {
	// could be multiple reports, however they must all be of the same type.
	for _, report := range reports {
		metricsCollections.Count++
		for _, stats := range report.GetData() {
			dimensionsChecksum := dimChecksum(stats)
			if _, ok := metricsCollections.Data[dimensionsChecksum]; !ok {
				metricsCollections.Data[dimensionsChecksum] = PerDimension{
					Dimensions:    stats.GetDimensions(),
					RunningSumMap: make(map[string]float64),
				}
			}

			if _, ok := metricsCollections.MetricsCount[dimensionsChecksum]; !ok {
				metricsCollections.MetricsCount[dimensionsChecksum] = PerDimension{
					Dimensions:    stats.GetDimensions(),
					RunningSumMap: make(map[string]float64),
				}
			}

			simpleMetrics := stats.GetSimplemetrics()

			for _, simpleMetric := range simpleMetrics {
				if metrics, ok := metricsCollections.Data[dimensionsChecksum].RunningSumMap[simpleMetric.Name]; ok {
					metricsCollections.MetricsCount[dimensionsChecksum].RunningSumMap[simpleMetric.Name]++
					metricsCollections.Data[dimensionsChecksum].RunningSumMap[simpleMetric.Name] = metrics + simpleMetric.GetValue()
				} else {
					metricsCollections.MetricsCount[dimensionsChecksum].RunningSumMap[simpleMetric.Name] = 1
					metricsCollections.Data[dimensionsChecksum].RunningSumMap[simpleMetric.Name] = simpleMetric.GetValue()
				}
			}
		}
	}

	return metricsCollections
}

func GenerateMetrics(metricsCollections Collections) []*proto.StatsEntity {
	results := make([]*proto.StatsEntity, 0, 200)

	for name, metricsPerDimension := range metricsCollections.Data {
		simpleMetrics := getAggregatedSimpleMetric(metricsCollections.MetricsCount[name], metricsPerDimension.RunningSumMap)
		results = append(results, NewStatsEntity(
			metricsPerDimension.Dimensions,
			simpleMetrics,
		))
	}

	return results
}

func getAggregatedSimpleMetric(count PerDimension, internalMap map[string]float64) (simpleMetrics []*proto.SimpleMetric) {
	variableMetrics := map[*regexp.Regexp]MetricsHandler{
		failsRegex: sum,
		freeRegex:  avg,
		reqsRegex:  sum,
		usedRegex:  avg,
	}

	calMap := GetCalculationMap()

	for name, value := range internalMap {
		if valueType, ok := calMap[name]; ok {
			var aggregatedValue float64
			switch valueType {
			case "sum":
				aggregatedValue = sum(value, int(count.RunningSumMap[name]))

			case "avg":
				aggregatedValue = avg(value, int(count.RunningSumMap[name]))

			case "boolean":
				aggregatedValue = boolean(value, int(count.RunningSumMap[name]))
			}

			// Only aggregate metrics when the aggregation method is defined
			simpleMetrics = append(simpleMetrics, &proto.SimpleMetric{
				Name:  name,
				Value: aggregatedValue,
			})
		} else {
			for reg, calculation := range variableMetrics {
				if reg.MatchString(name) {
					result := calculation(value, int(count.RunningSumMap[name]))

					simpleMetrics = append(simpleMetrics, &proto.SimpleMetric{
						Name:  name,
						Value: result,
					})
				}
			}
		}
	}

	return simpleMetrics
}

func sum(value float64, count int) float64 {
	// the value is already summed in collection
	return value
}

func avg(value float64, count int) float64 {
	if count > 0 {
		// the value is already summed in collection
		return value / float64(count)
	} else {
		return value
	}
}

// the return value is boolean in 1 or 0.
func boolean(value float64, count int) float64 {
	const ZERO, TEST, ONE float64 = 0.0, 0.5, 1.0

	floatCount := float64(count)
	if floatCount == ZERO {
		return value
	}

	// the value is already summed in collection
	average := value / floatCount
	if average > TEST {
		return ONE
	}

	return ZERO
}
