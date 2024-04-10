// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package prometheus

import (
	"math"
	"testing"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/stretchr/testify/require"
)

var (
	floatCounter = model.DataEntry{
		Name:        "go_memstats_alloc_bytes_total",
		Description: "Total number of bytes allocated, even if freed.",
		Type:        model.Counter,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name:   "go_memstats_alloc_bytes_total",
				Labels: nil,
				Value:  float64(130301312),
			},
		},
	}

	intCounter = model.DataEntry{
		Name:        "go_memstats_alloc_bytes_total",
		Description: "Total number of bytes allocated, even if freed.",
		Type:        model.Counter,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name:   "go_memstats_alloc_bytes_total",
				Labels: nil,
				Value:  int64(130301312),
			},
		},
	}

	//nolint: dupl
	floatHistogram = model.DataEntry{
		Name:        "prometheus_tsdb_compaction_chunk_range_seconds",
		Description: "Final time range of chunks on their first compaction",
		Type:        model.Histogram,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "100",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "400",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "1600",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "6400",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "25600",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "102400",
				},
				Value: float64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "409600",
				},
				Value: float64(1),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "1.6384e+06",
				},
				Value: float64(1),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "6.5536e+06",
				},
				Value: float64(21),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "2.62144e+07",
				},
				Value: float64(21),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "+Inf",
				},
				Value: float64(21),
			},
			{
				Name:   "prometheus_tsdb_compaction_chunk_range_seconds_sum",
				Labels: nil,
				Value:  float64(35685964),
			},
			{
				Name:   "prometheus_tsdb_compaction_chunk_range_seconds_count",
				Labels: nil,
				Value:  int64(21),
			},
		},
	}

	//nolint: dupl
	intHistogram = model.DataEntry{
		Name:        "prometheus_tsdb_compaction_chunk_range_seconds",
		Description: "Final time range of chunks on their first compaction",
		Type:        model.Histogram,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "100",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "400",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "1600",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "6400",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "25600",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "102400",
				},
				Value: int64(0),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "409600",
				},
				Value: int64(1),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "1.6384e+06",
				},
				Value: int64(1),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "6.5536e+06",
				},
				Value: int64(21),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "2.62144e+07",
				},
				Value: int64(21),
			},
			{
				Name: "prometheus_tsdb_compaction_chunk_range_seconds_bucket",
				Labels: map[string]string{
					"le": "+Inf",
				},
				Value: int64(21),
			},
			{
				Name:   "prometheus_tsdb_compaction_chunk_range_seconds_sum",
				Labels: nil,
				Value:  int64(35685964),
			},
			{
				Name:   "prometheus_tsdb_compaction_chunk_range_seconds_count",
				Labels: nil,
				Value:  int64(21),
			},
		},
	}
)

// nolint: dupl
func TestPrometheusConverter_FloatGauge(t *testing.T) {
	goVersion, err := helpers.GetGoVersion(t, 4)
	require.NoError(t, err)

	floatGauge := model.DataEntry{
		Name:        "go_info",
		Description: "Information about the Go environment.",
		Type:        model.Gauge,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "go_info",
				Labels: map[string]string{
					"version": goVersion,
				},
				Value: float64(1),
			},
		},
	}

	expData := []metricdata.DataPoint[float64]{
		{
			Attributes: attribute.NewSet(
				attribute.KeyValue{
					Key:   "version",
					Value: attribute.StringValue(goVersion),
				},
			),
			Value: float64(1),
		},
	}
	expMetricData := metricdata.Metrics{
		Name:        floatGauge.Name,
		Description: floatGauge.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(floatGauge)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Gauge[float64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.InEpsilon(t, expData[i].Value, act.Value, 0.0001)
	}
}

// nolint: dupl
func TestPrometheusConverter_IntGauge(t *testing.T) {
	goVersion, err := helpers.GetGoVersion(t, 4)
	require.NoError(t, err)

	intGauge := model.DataEntry{
		Name:        "go_info",
		Description: "Information about the Go environment.",
		Type:        model.Gauge,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "go_info",
				Labels: map[string]string{
					"version": goVersion,
				},
				Value: int64(1),
			},
		},
	}

	expData := []metricdata.DataPoint[int64]{
		{
			Attributes: attribute.NewSet(
				attribute.KeyValue{
					Key:   "version",
					Value: attribute.StringValue(goVersion),
				},
			),
			Value: int64(1),
		},
	}
	expMetricData := metricdata.Metrics{
		Name:        intGauge.Name,
		Description: intGauge.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(intGauge)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Gauge[int64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.Equal(t, expData[i].Value, act.Value)
	}
}

// nolint: dupl
func TestPrometheusConverter_FloatCounter(t *testing.T) {
	expData := []metricdata.DataPoint[float64]{
		{
			Attributes: *attribute.EmptySet(),
			Value:      float64(130301312),
		},
	}

	expPayload := metricdata.Sum[float64]{
		Temporality: metricdata.DeltaTemporality,
		IsMonotonic: true,
	}

	expMetricData := metricdata.Metrics{
		Name:        floatCounter.Name,
		Description: floatCounter.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(floatCounter)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Sum[float64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	assert.Equal(t, expPayload.Temporality, actualData.Temporality)
	assert.Equal(t, expPayload.IsMonotonic, actualData.IsMonotonic)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.InEpsilon(t, expData[i].Value, act.Value, 0.0001)
	}
}

// nolint: dupl
func TestPrometheusConverter_IntCounter(t *testing.T) {
	expData := []metricdata.DataPoint[int64]{
		{
			Attributes: *attribute.EmptySet(),
			Value:      int64(130301312),
		},
	}

	expPayload := metricdata.Sum[int64]{
		Temporality: metricdata.DeltaTemporality,
		IsMonotonic: true,
	}

	expMetricData := metricdata.Metrics{
		Name:        intCounter.Name,
		Description: intCounter.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(intCounter)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Sum[int64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	assert.Equal(t, expPayload.Temporality, actualData.Temporality)
	assert.Equal(t, expPayload.IsMonotonic, actualData.IsMonotonic)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.Equal(t, expData[i].Value, act.Value)
	}
}

// nolint: dupl
func TestPrometheusConverter_FloatHistogram(t *testing.T) {
	expData := []metricdata.HistogramDataPoint[float64]{
		{
			Attributes: *attribute.EmptySet(),
			BucketCounts: []uint64{
				0, 0, 0, 0, 0, 0, 1, 1, 21, 21, 21,
			},
			Bounds: []float64{
				100.0, 400.0, 1600.0, 6400.0, 25600.0, 102400.0, 409600.0,
				1638400.0, 6553600.0, 26214400.0, +math.Inf(1),
			},
			Sum:   float64(35685964),
			Count: uint64(21),
		},
	}

	expPayload := metricdata.Histogram[float64]{
		Temporality: metricdata.CumulativeTemporality,
	}

	expMetricData := metricdata.Metrics{
		Name:        floatHistogram.Name,
		Description: floatHistogram.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(floatHistogram)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Histogram[float64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	assert.Equal(t, expPayload.Temporality, actualData.Temporality)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.InEpsilon(t, expData[i].Sum, act.Sum, 0.0001)
		assert.Equal(t, expData[i].Count, act.Count)
		assert.Equal(t, expData[i].Bounds, act.Bounds)
		assert.Equal(t, expData[i].BucketCounts, act.BucketCounts)
	}
}

// nolint: dupl
func TestPrometheusConverter_IntHistogram(t *testing.T) {
	expData := []metricdata.HistogramDataPoint[int64]{
		{
			Attributes: *attribute.EmptySet(),
			BucketCounts: []uint64{
				0, 0, 0, 0, 0, 0, 1, 1, 21, 21, 21,
			},
			Bounds: []float64{
				100.0, 400.0, 1600.0, 6400.0, 25600.0, 102400.0,
				409600.0, 1638400.0, 6553600.0, 26214400.0, +math.Inf(1),
			},
			Sum:   int64(35685964),
			Count: uint64(21),
		},
	}

	expPayload := metricdata.Histogram[int64]{
		Temporality: metricdata.CumulativeTemporality,
	}

	expMetricData := metricdata.Metrics{
		Name:        intHistogram.Name,
		Description: intHistogram.Description,
		Unit:        "",
	}

	result, err := ConvertPrometheus(intHistogram)
	require.NoError(t, err)

	assert.Equal(t, expMetricData.Name, result.Name)
	assert.Equal(t, expMetricData.Description, result.Description)
	assert.Equal(t, expMetricData.Unit, result.Unit)

	actualData, ok := result.Data.(metricdata.Histogram[int64])
	assert.True(t, ok)
	assert.NotNil(t, actualData)

	assert.Equal(t, expPayload.Temporality, actualData.Temporality)

	for i, act := range actualData.DataPoints {
		assert.Equal(t, expData[i].Attributes, act.Attributes)
		assert.Equal(t, expData[i].Sum, act.Sum)
		assert.Equal(t, expData[i].Count, act.Count)
		assert.Equal(t, expData[i].Bounds, act.Bounds)
		assert.Equal(t, expData[i].BucketCounts, act.BucketCounts)
	}
}

func TestPrometheusConverter_Errors(t *testing.T) {
	goVersion, err := helpers.GetGoVersion(t, 4)
	require.NoError(t, err)

	t.Run("no-data-points", func(tt *testing.T) {
		input := testDataPoint(tt)
		res, conErr := ConvertPrometheus(input)
		require.NoError(tt, conErr)
		assert.Equal(tt, input.Name, res.Name)
		assert.Equal(tt, input.Description, res.Description)
		assert.Equal(tt, "", res.Unit)
		assert.Nil(tt, res.Data)
	})

	t.Run("unsupported-summary-type-returns-empty-data", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.Summary
		// Need to have at least one data point, so we don't return before parsing Type.
		input.Values = []model.DataPoint{
			{
				Name: "go_info",
				Labels: map[string]string{
					"version": goVersion,
				},
				Value: float64(1),
			},
		}

		res, conErr := ConvertPrometheus(input)
		require.NoError(tt, conErr)
		assert.Equal(tt, metricdata.Metrics{}, res)
	})

	t.Run("unsupported-point-value-type", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.UnknownInstrument
		input.Values = []model.DataPoint{
			{
				Name: "test-data-point",
				Labels: map[string]string{
					"test-label": "test-value",
				},
				Value: "not-a-valid-type",
			},
		}
		res, conErr := ConvertPrometheus(input)
		require.Error(tt, conErr)
		assert.Equal(tt, "could not convert data entry of value type [string] to OTel metric", conErr.Error())
		assert.Equal(tt, metricdata.Metrics{}, res)
	})

	t.Run("unexpected-counter-value-skipped", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.Counter
		input.Values = []model.DataPoint{
			{
				Name: "test-data-point-01",
				Labels: map[string]string{
					"test-label": "test-value",
				},
				Value: float64(32),
			},
			{
				Name: "test-data-point-02",
				Labels: map[string]string{
					"test-label": "test-value",
				},
				Value: "not-a-valid-type",
			},
		}

		res, convErr := ConvertPrometheus(input)
		require.NoError(tt, convErr)

		data, ok := res.Data.(metricdata.Sum[float64])
		require.True(t, ok)

		assert.Len(t, data.DataPoints, 1, "only one data point with valid value expected")
	})

	t.Run("unexpected-gauge-value-skipped", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.Gauge
		input.Values = []model.DataPoint{
			{
				Name: "go_info",
				Labels: map[string]string{
					"version": goVersion,
				},
				Value: float64(23),
			},
			{
				Name: "go_info",
				Labels: map[string]string{
					"version": goVersion,
				},
				Value: "not-a-valid-type",
			},
		}

		res, convErr := ConvertPrometheus(input)
		require.NoError(tt, convErr)

		data, ok := res.Data.(metricdata.Gauge[float64])
		require.True(t, ok)

		assert.Len(t, data.DataPoints, 1, "only one data point with valid value expected")
	})

	t.Run("unexpected-histogram-sum-value-returns-error", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.Histogram
		input.Values = []model.DataPoint{
			{
				Name: "test-histogram-point_bucket",
				Labels: map[string]string{
					"le": "100",
				},
				Value: float64(1),
			},
			{
				Name: "test-histogram-point_bucket",
				Labels: map[string]string{
					"le": "100",
				},
				Value: "not-a-valid-type",
			},
			{
				Name: "test-histogram-point_sum",
				Labels: map[string]string{
					"le": "110",
				},
				Value: "not-a-valid-type",
			},
			{
				Name: "test-histogram-point_count",
				Labels: map[string]string{
					"le": "120",
				},
				Value: "not-a-valid-type",
			},
		}

		_, err = ConvertPrometheus(input)
		require.Error(tt, err)
		assert.Equal(t, "could not convert data entry of value type [string] to OTel metric", err.Error())
	})

	t.Run("unsupported-instrument-type", func(tt *testing.T) {
		input := testDataPoint(tt)
		input.Type = model.UnknownInstrument
		input.Values = []model.DataPoint{
			{
				Name: "test-data-point-01",
				Labels: map[string]string{
					"test-label": "test-value",
				},
				Value: float64(32),
			},
		}

		res, convErr := ConvertPrometheus(input)
		require.Error(tt, convErr)
		assert.Equal(tt, "unhandled metrics conversion of type: unknown", convErr.Error())

		assert.Equal(tt, metricdata.Metrics{}, res)
	})
}

func testDataPoint(t *testing.T) model.DataEntry {
	t.Helper()
	return model.DataEntry{
		Name:        "test-entry",
		Description: "test description",
		Type:        model.Gauge,
		SourceType:  model.Prometheus,
		Values:      make([]model.DataPoint, 0),
	}
}
