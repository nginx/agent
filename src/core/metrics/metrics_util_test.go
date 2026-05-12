/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTimeMetrics_EmptyReturnsZero(t *testing.T) {
	for _, mt := range []string{"time", "count", "max", "median", "pctl95"} {
		t.Run(mt, func(t *testing.T) {
			assert.InDelta(t, 0.0, GetTimeMetrics(nil, mt), 0)
			assert.InDelta(t, 0.0, GetTimeMetrics([]float64{}, mt), 0)
		})
	}
}

func TestGetTimeMetrics_UnknownMetricTypeReturnsZero(t *testing.T) {
	got := GetTimeMetrics([]float64{1, 2, 3}, "not-a-real-type")
	assert.InDelta(t, 0.0, got, 0)
}

func TestGetTimeMetrics_TimeAverage(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  float64
	}{
		{"single value", []float64{4.0}, 4.0},
		{"simple average", []float64{2.0, 4.0, 6.0}, 4.0},
		{"average of two", []float64{1.0, 2.0}, 1.5},
		{"rounded to 3dp before division", []float64{0.111, 0.222, 0.333}, 0.222},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTimeMetrics(tt.input, "time")
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestGetTimeMetrics_Count(t *testing.T) {
	got := GetTimeMetrics([]float64{1.5, 2.5, 3.5, 4.5}, "count")
	assert.InDelta(t, 4.0, got, 0)
}

func TestGetTimeMetrics_Max(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  float64
	}{
		{"single", []float64{42.0}, 42.0},
		{"already sorted", []float64{1, 2, 3, 4, 5}, 5},
		{"unsorted", []float64{3, 1, 4, 1, 5, 9, 2, 6}, 9},
		{"negatives", []float64{-3, -1, -2}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]float64{}, tt.input...)
			got := GetTimeMetrics(input, "max")
			assert.InDelta(t, tt.want, got, 0)
		})
	}
}

func TestGetTimeMetrics_Median(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  float64
	}{
		{"odd count", []float64{1, 2, 3, 4, 5}, 3},
		{"odd count unsorted", []float64{5, 1, 3, 4, 2}, 3},
		{"even count", []float64{1, 2, 3, 4}, 2.5},
		{"even count unsorted", []float64{4, 1, 3, 2}, 2.5},
		{"single value", []float64{7.5}, 7.5},
		{"two values", []float64{2, 4}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]float64{}, tt.input...)
			got := GetTimeMetrics(input, "median")
			assert.InDelta(t, tt.want, got, 0)
		})
	}
}

func TestGetTimeMetrics_Pctl95(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  float64
	}{
		{"twenty values", makeRange(1, 20), 19},
		{"ten values - banker rounds up", makeRange(1, 10), 10},
		{"two values", []float64{1, 2}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]float64{}, tt.input...)
			got := GetTimeMetrics(input, "pctl95")
			assert.InDelta(t, tt.want, got, 0)
		})
	}
}

func makeRange(lo, hi int) []float64 {
	out := make([]float64, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		out = append(out, float64(i))
	}
	return out
}

func TestNewStatsEntityWrapper_PreservesType(t *testing.T) {
	dims := []*proto.Dimension{{Name: "host", Value: "h1"}}
	samples := []*proto.SimpleMetric{{Name: "cpu", Value: 0.5}}

	w := NewStatsEntityWrapper(dims, samples, proto.MetricsReport_SYSTEM)

	require.NotNil(t, w)
	assert.Equal(t, proto.MetricsReport_SYSTEM, w.Type)
	require.NotNil(t, w.Data)
	assert.Equal(t, dims, w.Data.Dimensions)
	assert.Equal(t, samples, w.Data.Simplemetrics)
	assert.NotNil(t, w.Data.Timestamp)
}

func TestNewStatsEntity_PopulatesFields(t *testing.T) {
	dims := []*proto.Dimension{{Name: "host", Value: "h1"}}
	samples := []*proto.SimpleMetric{{Name: "cpu", Value: 0.5}}

	se := NewStatsEntity(dims, samples)

	require.NotNil(t, se)
	assert.Equal(t, dims, se.Dimensions)
	assert.Equal(t, samples, se.Simplemetrics)
	assert.NotNil(t, se.Timestamp)
}

func TestGetCalculationMap_ContainsExpectedKeys(t *testing.T) {
	m := GetCalculationMap()
	require.NotNil(t, m)

	cases := map[string]string{
		"system.cpu.user":         "avg",
		"system.io.iops_r":        "sum",
		"nginx.status":            "boolean",
		"nginx.http.status.2xx":   "sum",
		"nginx.http.request.time": "avg",
		"plus.http.request.count": "sum",
		"container.cpu.cores":     "avg",
	}

	for key, want := range cases {
		got, ok := m[key]
		assert.True(t, ok, "key %q missing from calculation map", key)
		assert.Equal(t, want, got, "key %q has wrong calc type", key)
	}
}

func TestGetCalculationMap_ReturnsConsistentSnapshots(t *testing.T) {
	a := GetCalculationMap()
	b := GetCalculationMap()
	assert.Equal(t, len(a), len(b))
}

func TestGenerateMetricsReportBundle_NilWhenEmpty(t *testing.T) {
	got := GenerateMetricsReportBundle(nil)
	assert.Nil(t, got)

	got = GenerateMetricsReportBundle([]*StatsEntityWrapper{})
	assert.Nil(t, got)
}

func TestGenerateMetricsReportBundle_SkipsNilEntities(t *testing.T) {
	got := GenerateMetricsReportBundle([]*StatsEntityWrapper{nil, nil})
	assert.Nil(t, got, "all-nil input should produce a nil bundle")
}

func TestGenerateMetricsReportBundle_SkipsEntitiesWithNilData(t *testing.T) {
	entities := []*StatsEntityWrapper{
		{Type: proto.MetricsReport_SYSTEM, Data: nil},
	}
	got := GenerateMetricsReportBundle(entities)
	assert.Nil(t, got)
}

func TestGenerateMetricsReportBundle_GroupsByType(t *testing.T) {
	systemDims := []*proto.Dimension{{Name: "host", Value: "h"}}
	systemSamples := []*proto.SimpleMetric{{Name: "cpu", Value: 1}}
	instanceDims := []*proto.Dimension{{Name: "instance", Value: "i"}}
	instanceSamples := []*proto.SimpleMetric{{Name: "rps", Value: 2}}

	entities := []*StatsEntityWrapper{
		NewStatsEntityWrapper(systemDims, systemSamples, proto.MetricsReport_SYSTEM),
		NewStatsEntityWrapper(systemDims, systemSamples, proto.MetricsReport_SYSTEM),
		NewStatsEntityWrapper(instanceDims, instanceSamples, proto.MetricsReport_INSTANCE),
	}

	got := GenerateMetricsReportBundle(entities)
	require.NotNil(t, got)

	bundle, ok := got.(*MetricsReportBundle)
	require.True(t, ok, "expected *MetricsReportBundle, got %T", got)
	require.Len(t, bundle.Data, 2, "expected one MetricsReport per type")

	byType := make(map[proto.MetricsReport_Type]int)
	for _, r := range bundle.Data {
		byType[r.Type] = len(r.Data)
		assert.NotNil(t, r.Meta)
		assert.NotNil(t, r.Meta.Timestamp)
	}
	assert.Equal(t, 2, byType[proto.MetricsReport_SYSTEM])
	assert.Equal(t, 1, byType[proto.MetricsReport_INSTANCE])
}

func TestGetTimeMetrics_DoesNotMutateInputForTime(t *testing.T) {
	original := []float64{3, 1, 4, 1, 5, 9, 2, 6}
	copyIn := append([]float64{}, original...)

	_ = GetTimeMetrics(copyIn, "time")
	assert.Equal(t, original, copyIn)

	_ = GetTimeMetrics(copyIn, "count")
	assert.Equal(t, original, copyIn)
}
