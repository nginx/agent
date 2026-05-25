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

func TestGetTimeMetrics(t *testing.T) {
	tests := []struct {
		name   string
		input  []float64
		metric string
		want   float64
		delta  float64
	}{
		{"empty-time", nil, "time", 0, 0},
		{"empty-count", nil, "count", 0, 0},
		{"empty-max", nil, "max", 0, 0},
		{"empty-median", nil, "median", 0, 0},
		{"empty-pctl95", nil, "pctl95", 0, 0},
		{"empty-slice-time", []float64{}, "time", 0, 0},
		{"unknown-metric-type", []float64{1, 2, 3}, "not-a-real-type", 0, 0},
		{"time-single", []float64{4.0}, "time", 4.0, 1e-9},
		{"time-simple-average", []float64{2.0, 4.0, 6.0}, "time", 4.0, 1e-9},
		{"time-average-of-two", []float64{1.0, 2.0}, "time", 1.5, 1e-9},
		{"time-rounded-3dp", []float64{0.111, 0.222, 0.333}, "time", 0.222, 1e-9},
		{"count", []float64{1.5, 2.5, 3.5, 4.5}, "count", 4.0, 0},
		{"max-single", []float64{42.0}, "max", 42.0, 0},
		{"max-sorted", []float64{1, 2, 3, 4, 5}, "max", 5, 0},
		{"max-unsorted", []float64{3, 1, 4, 1, 5, 9, 2, 6}, "max", 9, 0},
		{"max-negatives", []float64{-3, -1, -2}, "max", -1, 0},
		{"median-odd", []float64{1, 2, 3, 4, 5}, "median", 3, 0},
		{"median-odd-unsorted", []float64{5, 1, 3, 4, 2}, "median", 3, 0},
		{"median-even", []float64{1, 2, 3, 4}, "median", 2.5, 0},
		{"median-even-unsorted", []float64{4, 1, 3, 2}, "median", 2.5, 0},
		{"median-single", []float64{7.5}, "median", 7.5, 0},
		{"median-two", []float64{2, 4}, "median", 3, 0},
		{"pctl95-twenty", makeRange(1, 20), "pctl95", 19, 0},
		{"pctl95-ten", makeRange(1, 10), "pctl95", 10, 0},
		{"pctl95-two", []float64{1, 2}, "pctl95", 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := append([]float64{}, tt.input...)
			got := GetTimeMetrics(input, tt.metric)
			assert.InDelta(t, tt.want, got, tt.delta, "Test %q failed", tt.name)
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

func TestGenerateMetricsReportBundles(t *testing.T) {
	systemDims := []*proto.Dimension{{Name: "host", Value: "h"}}
	systemSamples := []*proto.SimpleMetric{{Name: "cpu", Value: 1}}
	instanceDims := []*proto.Dimension{{Name: "instance", Value: "i"}}
	instanceSamples := []*proto.SimpleMetric{{Name: "rps", Value: 2}}

	tests := []struct {
		name     string
		input    []*StatsEntityWrapper
		wantNil  bool
		wantType map[proto.MetricsReport_Type]int // type to count
	}{
		{"NilWhenEmpty-nil", nil, true, nil},
		{"NilWhenEmpty-empty", []*StatsEntityWrapper{}, true, nil},
		{"SkipsNilEntities", []*StatsEntityWrapper{nil, nil}, true, nil},
		{"SkipsEntitiesWithNilData", []*StatsEntityWrapper{{Type: proto.MetricsReport_SYSTEM, Data: nil}}, true, nil},
		{
			"GroupsByType",
			[]*StatsEntityWrapper{
				NewStatsEntityWrapper(systemDims, systemSamples, proto.MetricsReport_SYSTEM),
				NewStatsEntityWrapper(systemDims, systemSamples, proto.MetricsReport_SYSTEM),
				NewStatsEntityWrapper(instanceDims, instanceSamples, proto.MetricsReport_INSTANCE),
			},
			false,
			map[proto.MetricsReport_Type]int{
				proto.MetricsReport_SYSTEM:   2,
				proto.MetricsReport_INSTANCE: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateMetricsReportBundle(tt.input)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				bundle, ok := got.(*MetricsReportBundle)
				require.True(t, ok, "expected *MetricsReportBundle, got %T", got)
				require.Len(t, bundle.Data, len(tt.wantType), "expected one MetricsReport per type")
				byType := make(map[proto.MetricsReport_Type]int)
				for _, r := range bundle.Data {
					byType[r.Type] = len(r.Data)
					assert.NotNil(t, r.Meta)
					assert.NotNil(t, r.Meta.Timestamp)
				}
				for typ, count := range tt.wantType {
					assert.Equal(t, count, byType[typ], "Test %q failed", tt.name)
				}
			}
		})
	}
}

func TestGetTimeMetrics_DoesNotMutateInputForTime(t *testing.T) {
	original := []float64{3, 1, 4, 1, 5, 9, 2, 6}
	copyIn := append([]float64{}, original...)

	_ = GetTimeMetrics(copyIn, "time")
	assert.Equal(t, original, copyIn)

	_ = GetTimeMetrics(copyIn, "count")
	assert.Equal(t, original, copyIn)
}
