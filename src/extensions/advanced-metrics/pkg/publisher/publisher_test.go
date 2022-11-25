/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package publisher

import (
	"context"
	"strconv"
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher/mocks"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/stretchr/testify/assert"
)

func TestPublisher(t *testing.T) {
	tests := []struct {
		name              string
		schema            *schema.Schema
		samples           map[string]*sample.Sample
		expectedMetrics   []*MetricSet
		dimensionsLookups map[int]map[int]string
	}{
		{
			name: "single metric with single dimension",
			schema: schema.NewSchema([]*schema.Field{
				schema.NewDimensionField("dim1", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric1"),
			}...),
			samples: map[string]*sample.Sample{
				"s1": testSample(t, []float64{1}, []int{1}),
				"s2": testSample(t, []float64{2}, []int{2}),
			},
			dimensionsLookups: map[int]map[int]string{
				0: {
					1: "dim1val1",
					2: "dim1val2",
				},
			},
			expectedMetrics: []*MetricSet{
				{
					Dimensions: []Dimension{
						{
							Name:  "dim1",
							Value: "dim1val1",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  1,
								Min:   1,
								Max:   1,
								Sum:   1,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim1",
							Value: "dim1val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  2,
								Min:   2,
								Max:   2,
								Sum:   2,
							},
						},
					},
				},
			},
		},
		{
			name: "dimension with transform function",
			schema: schema.NewSchema([]*schema.Field{
				schema.NewDimensionField("dim1", 0, schema.WithKeyBitSize(8), schema.WithTransformFunction(&schema.DimensionTransformFunction{
					FromLookupCodeToValue: func(code int) (string, error) {
						return strconv.Itoa(code), nil
					},
				})),
				schema.NewMetricField("metric1"),
			}...),
			samples: map[string]*sample.Sample{
				"s1": testSample(t, []float64{1}, []int{1}),
			},
			dimensionsLookups: map[int]map[int]string{},
			expectedMetrics: []*MetricSet{
				{
					Dimensions: []Dimension{
						{
							Name:  "dim1",
							Value: "1",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  1,
								Min:   1,
								Max:   1,
								Sum:   1,
							},
						},
					},
				},
			},
		},
		{
			name: "multiple metric with multiple dimension",
			schema: schema.NewSchema([]*schema.Field{
				schema.NewDimensionField("dim1", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric1"),
				schema.NewDimensionField("dim2", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric2"),
			}...),
			samples: map[string]*sample.Sample{
				"s1": testSample(t, []float64{1, 11}, []int{1, 11}),
				"s2": testSample(t, []float64{2, 12}, []int{1, 12}),
				"s3": testSample(t, []float64{3, 13}, []int{2, 12}),
			},
			dimensionsLookups: map[int]map[int]string{
				0: {
					1: "dim1val1",
					2: "dim1val2",
				},
				1: {
					11: "dim2val1",
					12: "dim2val2",
				},
			},
			expectedMetrics: []*MetricSet{
				{
					Dimensions: []Dimension{
						{
							Name:  "dim1",
							Value: "dim1val1",
						},
						{
							Name:  "dim2",
							Value: "dim2val1",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  1,
								Min:   1,
								Max:   1,
								Sum:   1,
							},
						},
						{
							Name: "metric2",
							Values: sample.Metric{
								Count: 1,
								Last:  11,
								Min:   11,
								Max:   11,
								Sum:   11,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim1",
							Value: "dim1val1",
						},
						{

							Name:  "dim2",
							Value: "dim2val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  2,
								Min:   2,
								Max:   2,
								Sum:   2,
							},
						},
						{
							Name: "metric2",
							Values: sample.Metric{
								Count: 1,
								Last:  12,
								Min:   12,
								Max:   12,
								Sum:   12,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim1",
							Value: "dim1val2",
						},
						{

							Name:  "dim2",
							Value: "dim2val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  3,
								Min:   3,
								Max:   3,
								Sum:   3,
							},
						},
						{
							Name: "metric2",
							Values: sample.Metric{
								Count: 1,
								Last:  13,
								Min:   13,
								Max:   13,
								Sum:   13,
							},
						},
					},
				},
			},
		},
		{
			name: "multiple metric with multiple dimension, with different set of metrics",
			schema: schema.NewSchema([]*schema.Field{
				schema.NewDimensionField("dim1", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric1"),
				schema.NewDimensionField("dim2", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric2"),
			}...),
			samples: map[string]*sample.Sample{
				"s1": testSample(t, []float64{1}, []int{1, 11}),
				"s2": testSample(t, []float64{2}, []int{1, 12}),
				"s3": testSample(t, []float64{3, 13}, []int{2, 12}),
			},
			dimensionsLookups: map[int]map[int]string{
				0: {
					1: "dim1val1",
					2: "dim1val2",
				},
				1: {
					11: "dim2val1",
					12: "dim2val2",
				},
			},
			expectedMetrics: []*MetricSet{
				{
					Dimensions: []Dimension{
						{
							Name:  "dim1",
							Value: "dim1val1",
						},
						{
							Name:  "dim2",
							Value: "dim2val1",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  1,
								Min:   1,
								Max:   1,
								Sum:   1,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim1",
							Value: "dim1val1",
						},
						{

							Name:  "dim2",
							Value: "dim2val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  2,
								Min:   2,
								Max:   2,
								Sum:   2,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim1",
							Value: "dim1val2",
						},
						{

							Name:  "dim2",
							Value: "dim2val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  3,
								Min:   3,
								Max:   3,
								Sum:   3,
							},
						},
						{
							Name: "metric2",
							Values: sample.Metric{
								Count: 1,
								Last:  13,
								Min:   13,
								Max:   13,
								Sum:   13,
							},
						},
					},
				},
			},
		},
		{
			name: "multiple metric with multiple dimension, with NA dimensions",
			schema: schema.NewSchema([]*schema.Field{
				schema.NewDimensionField("dim1", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric1"),
				schema.NewDimensionField("dim2", 0, schema.WithKeyBitSize(8)),
				schema.NewMetricField("metric2"),
			}...),
			samples: map[string]*sample.Sample{
				"s1": testSample(t, []float64{1}, []int{1, 11}),
				"s2": testSample(t, []float64{2}, []int{lookup.LookupNACode, 12}),
			},
			dimensionsLookups: map[int]map[int]string{
				0: {
					1: "dim1val1",
					2: "dim1val2",
				},
				1: {
					11: "dim2val1",
					12: "dim2val2",
				},
			},
			expectedMetrics: []*MetricSet{
				{
					Dimensions: []Dimension{
						{
							Name:  "dim1",
							Value: "dim1val1",
						},
						{
							Name:  "dim2",
							Value: "dim2val1",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  1,
								Min:   1,
								Max:   1,
								Sum:   1,
							},
						},
					},
				},
				{
					Dimensions: []Dimension{
						{

							Name:  "dim2",
							Value: "dim2val2",
						},
					},
					Metrics: []Metric{
						{
							Name: "metric1",
							Values: sample.Metric{
								Count: 1,
								Last:  2,
								Min:   2,
								Max:   2,
								Sum:   2,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outChannel := make(chan []*MetricSet, 1)
			publisher := New(outChannel, test.schema)
			err := publisher.Publish(
				context.Background(),
				&mocks.LookupSetStub{Lookups: test.dimensionsLookups},
				&mocks.PriorityTableStub{SamplesMap: test.samples})
			assert.NoError(t, err)
			metrics := <-outChannel
			assert.ElementsMatch(t, test.expectedMetrics, metrics)

		})
	}
}

func testSample(t *testing.T, metrics []float64, dimensionsCodes []int) *sample.Sample {
	s := sample.NewSample(len(dimensionsCodes)*8, len(metrics))
	for _, c := range dimensionsCodes {
		err := s.Key().AddKeyPart(c, 8)
		assert.NoError(t, err)
	}

	for i, m := range metrics {
		err := s.SetMetric(i, m)
		assert.NoError(t, err)
	}

	return &s
}
