package validator

import (
	"sort"
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	"github.com/stretchr/testify/assert"
)

type ExpectedMetric struct {
	Name       string
	MinRange   Range
	MaxRange   Range
	CountRange Range
	SumRange   Range
}

type Range struct {
	Low  float64
	High float64
}

func AssertMetricSetEqual(t *testing.T, expectedMetrics []ExpectedMetric, expectedDimensions []publisher.Dimension, actuallMetrics *publisher.MetricSet) {
	assertDimensionsEqual(t, expectedDimensions, actuallMetrics)
	assertMetricsEqual(t, expectedMetrics, actuallMetrics.Metrics)
}

func assertMetricsEqual(t *testing.T, expectedMetrics []ExpectedMetric,
	actualMetrics []publisher.Metric) {

	actualMetricsMap := make(map[string]publisher.Metric)

	for _, actualMetric := range actualMetrics {
		actualMetricsMap[actualMetric.Name] = actualMetric
	}

	assert.Len(t, actualMetrics, len(expectedMetrics))

	for _, expectedMetric := range expectedMetrics {
		actualMetric, ok := actualMetricsMap[expectedMetric.Name]
		assert.Truef(t, ok, "expected metric '%s' not found", expectedMetric.Name)
		assert.LessOrEqual(t, expectedMetric.MinRange.Low, actualMetric.Values.Min, expectedMetric.Name)
		assert.GreaterOrEqual(t, expectedMetric.MinRange.High, actualMetric.Values.Min, expectedMetric.Name)

		assert.LessOrEqual(t, expectedMetric.MaxRange.Low, actualMetric.Values.Max, expectedMetric.Name)
		assert.GreaterOrEqual(t, expectedMetric.MaxRange.High, actualMetric.Values.Max, expectedMetric.Name)

		assert.LessOrEqual(t, expectedMetric.CountRange.Low, actualMetric.Values.Count, expectedMetric.Name)
		assert.GreaterOrEqual(t, expectedMetric.CountRange.High, actualMetric.Values.Count, expectedMetric.Name)

		assert.LessOrEqual(t, expectedMetric.SumRange.Low, actualMetric.Values.Sum, expectedMetric.Name)
		assert.GreaterOrEqual(t, expectedMetric.SumRange.High, actualMetric.Values.Sum, expectedMetric.Name)
	}
}

func assertDimensionsEqual(t *testing.T, expectedDimensions []publisher.Dimension, actuallMetrics *publisher.MetricSet) {
	sort.Slice(actuallMetrics.Dimensions, func(i, j int) bool { return actuallMetrics.Dimensions[i].Name < actuallMetrics.Dimensions[j].Name })
	sort.Slice(expectedDimensions, func(i, j int) bool { return expectedDimensions[i].Name < expectedDimensions[j].Name })

	assert.Equal(t, expectedDimensions, actuallMetrics.Dimensions)
}
