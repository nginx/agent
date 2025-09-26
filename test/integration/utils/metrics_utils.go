// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package utils

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

// SumMetricFamily retrieves the sum of the metric values in a MetricFamily
func SumMetricFamily(metricFamily *dto.MetricFamily) float64 {
	var total float64
	for _, metric := range metricFamily.GetMetric() {
		if value := metricValue(metricFamily, metric); value != nil {
			total += *value
		}
	}

	return total
}

// SumMetricFamilyLabel retrieves the sum of the metric values for a specific label in a MetricFamily
func SumMetricFamilyLabel(metricFamily *dto.MetricFamily, key, val string) float64 {
	var total float64
	for _, metric := range metricFamily.GetMetric() {
		labels := make(map[string]string)
		for _, labelPair := range metric.GetLabel() {
			labels[labelPair.GetName()] = labelPair.GetValue()
		}
		if labels[key] != val {
			continue
		}
		if value := metricValue(metricFamily, metric); value != nil {
			total += *value
		}
	}

	return total
}

// checkSingleMetricValue compares a metric's current value against a baseline value
func checkSingleMetricValue(t *testing.T, family *dto.MetricFamily, baselineValue float64) (float64, bool) {
	t.Helper()
	metric := SumMetricFamily(family)

	return metric, metric != baselineValue
}

// checkLabeledMetricValue compares labeled metrics' current values against a set of baseline values
func checkLabeledMetricValue(t *testing.T, family *dto.MetricFamily, labelKey string,
	labelValues []string, baselineValues []float64,
) ([]float64, bool) {
	t.Helper()
	results := make([]float64, len(baselineValues))
	allDifferent := true

	for val := range labelValues {
		metric := SumMetricFamilyLabel(family, labelKey, labelValues[val])
		results[val] = metric
		if metric == baselineValues[val] {
			allDifferent = false
		}
	}

	return results, allDifferent && len(results) > 0
}

func metricValue(metricFamily *dto.MetricFamily, metric *dto.Metric) *float64 {
	switch metricFamily.GetType() {
	case dto.MetricType_COUNTER:
		return counterValue(metric)
	case dto.MetricType_GAUGE:
		return gaugeValue(metric)
	case dto.MetricType_SUMMARY:
		return summaryValue(metric)
	case dto.MetricType_UNTYPED:
		return untypedValue(metric)
	case dto.MetricType_HISTOGRAM, dto.MetricType_GAUGE_HISTOGRAM:
		return histogramValue(metric)
	}

	return nil
}

func counterValue(metric *dto.Metric) *float64 {
	if counter := metric.GetCounter(); counter != nil {
		val := counter.GetValue()
		return &val
	}

	return nil
}

func gaugeValue(metric *dto.Metric) *float64 {
	if gauge := metric.GetGauge(); gauge != nil {
		val := gauge.GetValue()
		return &val
	}

	return nil
}

func summaryValue(metric *dto.Metric) *float64 {
	if summary := metric.GetSummary(); summary != nil {
		val := summary.GetSampleSum()
		return &val
	}

	return nil
}

func untypedValue(metric *dto.Metric) *float64 {
	if untyped := metric.GetUntyped(); untyped != nil {
		val := untyped.GetValue()
		return &val
	}

	return nil
}

func histogramValue(metric *dto.Metric) *float64 {
	if histogram := metric.GetHistogram(); histogram != nil {
		val := histogram.GetSampleSum()
		return &val
	}

	return nil
}
