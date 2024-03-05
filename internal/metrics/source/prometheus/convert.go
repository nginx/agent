// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package prometheus

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/nginx/agent/v3/internal/model"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	bucketSuffix = "_bucket"
	sumSuffix    = "_sum"
	countSuffix  = "_count"
)

type InstrumentPointConverter[N float64 | int64] func(input model.DataEntry) metricdata.Aggregation

// ConvertPrometheus converts metrics data points from our internal model to OTel format.
func ConvertPrometheus(input model.DataEntry) (metricdata.Metrics, error) {
	result := metricdata.Metrics{
		Name:        input.Name,
		Description: input.Description,
	}

	if len(input.Values) == 0 {
		slog.Warn("Received metric with no data points", "metric_name", input.Name)
		return result, nil
	}

	switch getSample(input).Value.(type) {
	case int64:
		return toInstrumentDataPoint[int64](input, result)
	case float64:
		return toInstrumentDataPoint[float64](input, result)
	default:
		return metricdata.Metrics{}, fmt.Errorf(
			"could not convert data entry of value type [%T] to OTel metric", getSample(input).Value,
		)
	}
}

// Gets a data point that has the correct data type for the instrument type. Histogram is currently the only exception.
func getSample(de model.DataEntry) model.DataPoint {
	if de.Type == model.Histogram {
		for _, v := range de.Values {
			if strings.Contains(v.Name, sumSuffix) {
				return v
			}
		}
	}

	return de.Values[0]
}

func toInstrumentDataPoint[N float64 | int64](
	input model.DataEntry, base metricdata.Metrics,
) (metricdata.Metrics, error) {
	switch input.Type {
	case model.Counter:
		base.Data = toCounter[N](input)
	case model.Gauge:
		base.Data = toGauge[N](input)
	case model.Histogram:
		base.Data = toHistogramDataPoint[N](input)
	case model.Summary:
		slog.Debug("Unhandled metrics conversion of type 'summary': not supported yet")
		return metricdata.Metrics{}, nil
	case model.UnknownInstrument:
		fallthrough
	default:
		slog.Debug("Unhandled metrics conversion", "type", input.Type)
		return metricdata.Metrics{}, fmt.Errorf("unhandled metrics conversion of type: %s", input.Type)
	}

	return base, nil
}

func toCounter[N float64 | int64](input model.DataEntry) metricdata.Sum[N] {
	dataPoints := make([]metricdata.DataPoint[N], 0)

	for _, point := range input.Values {
		metricAttributes := make([]attribute.KeyValue, 0)
		for labelKey, labelValue := range point.Labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{
				Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue),
			})
		}

		value, ok := point.Value.(N)
		if !ok {
			// Do not return error for malformed data but instead ignore the data point.
			slog.Debug("Could not cast metric data point value to correct type: discarding data point",
				"point_name", point.Name, "value", point.Value)

			continue
		}

		dataPoints = append(dataPoints, metricdata.DataPoint[N]{
			Attributes: attribute.NewSet(metricAttributes...),
			Time:       time.Now(),
			Value:      value,
		})
	}

	return metricdata.Sum[N]{
		DataPoints:  dataPoints,
		Temporality: metricdata.DeltaTemporality,
		IsMonotonic: true,
	}
}

func toGauge[N float64 | int64](input model.DataEntry) metricdata.Gauge[N] {
	dataPoints := make([]metricdata.DataPoint[N], 0)

	for _, point := range input.Values {
		metricAttributes := make([]attribute.KeyValue, 0)
		for labelKey, labelValue := range point.Labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{
				Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue),
			})
		}

		value, ok := point.Value.(N)
		if !ok {
			// Do not return error for malformed data but instead ignore the data point.
			slog.Debug("Could not cast metric data point value to correct type: discarding data point",
				"point_name", point.Name, "value", point.Value)

			continue
		}

		dataPoints = append(dataPoints, metricdata.DataPoint[N]{
			Attributes: attribute.NewSet(metricAttributes...),
			Time:       time.Now(),
			Value:      value,
		})
	}

	return metricdata.Gauge[N]{
		DataPoints: dataPoints,
	}
}

func toHistogramDataPoint[N float64 | int64](input model.DataEntry) metricdata.Histogram[N] {
	histogram := metricdata.HistogramDataPoint[N]{
		Bounds:       []float64{},
		BucketCounts: []uint64{},
		StartTime:    time.Now(),
		Time:         time.Now(),
	}

	for _, point := range input.Values {
		processPoint(point, &histogram)
	}

	return metricdata.Histogram[N]{
		DataPoints: []metricdata.HistogramDataPoint[N]{histogram},
		// Not sure if Prometheus histograms are deltas or cumulatives.
		Temporality: metricSdk.DefaultTemporalitySelector(metricSdk.InstrumentKindHistogram),
	}
}

func processPoint[N float64 | int64](point model.DataPoint, histogram *metricdata.HistogramDataPoint[N]) {
	metricAttributes, bound := parseHistogramLabels(point)

	if strings.HasSuffix(point.Name, bucketSuffix) && bound != "" {
		parseHistogramBucket(point, histogram, bound)
	} else if strings.HasSuffix(point.Name, sumSuffix) {
		parseHistogramSum(point, histogram)
	} else if strings.HasSuffix(point.Name, countSuffix) {
		parseHistogramCount(point, histogram, metricAttributes)
	}
}

func parseHistogramLabels(point model.DataPoint) ([]attribute.KeyValue, string) {
	var bound string

	metricAttributes := make([]attribute.KeyValue, 0)
	for labelKey, labelValue := range point.Labels {
		if labelKey == "le" {
			bound = labelValue
		}
		metricAttributes = append(metricAttributes, attribute.KeyValue{
			Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue),
		})
	}

	return metricAttributes, bound
}

func parseHistogramBucket[N float64 | int64](
	point model.DataPoint, hist *metricdata.HistogramDataPoint[N], bound string,
) {
	value, err := toUint64(point.Value)
	if err != nil {
		slog.Debug("Could not cast metric data point value to correct type: discarding data point",
			"point_name", point.Name, "value", point.Value, "error", err)

		return
	}
	hist.BucketCounts = append(hist.BucketCounts, value)
	boundValue, _ := strconv.ParseFloat(bound, 64)
	hist.Bounds = append(hist.Bounds, boundValue)
}

func parseHistogramSum[N float64 | int64](point model.DataPoint, hist *metricdata.HistogramDataPoint[N]) {
	switch castVal := point.Value.(type) {
	case N:
		hist.Sum = castVal
	default:
		slog.Debug("Could not cast histogram sum value to correct type: discarding data point",
			"point_name", point.Name, "value", point.Value)
	}
}

func parseHistogramCount[N float64 | int64](
	point model.DataPoint, hist *metricdata.HistogramDataPoint[N], attrs []attribute.KeyValue,
) {
	value, err := toUint64(point.Value)
	if err != nil {
		slog.Debug("Could not cast metric data point value to correct type: discarding data point",
			"point_name", point.Name, "value", point.Value, "error", err)

		return
	}
	hist.Count = value
	hist.Attributes = attribute.NewSet(attrs...)
}

// Assumes that `input` is of type `float64` or `int64`.
func toUint64(input any) (uint64, error) {
	var value uint64
	switch p := input.(type) {
	case int64:
		value = uint64(p)
	case float64:
		value = uint64(p)
	default:
		return 0, fmt.Errorf("histogram value of unsupported data type %T", input)
	}

	return value, nil
}
