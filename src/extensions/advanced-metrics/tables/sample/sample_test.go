/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sample

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	keySize     = 10
	metricsSize = 10
)

func TestNewSample(t *testing.T) {
	sample := NewSample(keySize, metricsSize)
	assert.Equal(t, 1, sample.HitCount())
	assert.Equal(t, metricsSize, len(sample.Metrics()))
	assert.Equal(t, keySize, len(sample.key.AsByteKey()))
}

func TestSampleSetMetric(t *testing.T) {
	sample := NewSample(keySize, metricsSize)

	metricIndex := keySize / 2
	metricValue := float64(42)
	err := sample.SetMetric(metricIndex, metricValue)
	assert.NoError(t, err)

	metric, err := sample.Metric(metricIndex)
	assert.NoError(t, err)
	assert.Equal(t, NewMetric(metricValue), metric)

	metrics := sample.Metrics()
	assert.Equal(t, NewMetric(metricValue), metrics[metricIndex])
}

func TestSampleAddSample(t *testing.T) {
	sample := NewSample(keySize, 2)
	metricValue := float64(42)
	metricValue2 := float64(43)
	err := sample.SetMetric(0, metricValue)
	assert.NoError(t, err)
	err = sample.SetMetric(1, metricValue2)
	assert.NoError(t, err)

	sample2 := NewSample(keySize, 2)
	metric2Value := float64(12)
	metric2Value2 := float64(13)
	err = sample2.SetMetric(0, metric2Value)
	assert.NoError(t, err)
	err = sample2.SetMetric(1, metric2Value2)
	assert.NoError(t, err)

	err = sample.AddSample(&sample2)
	assert.NoError(t, err)

	metrics := sample.Metrics()
	assert.Equal(t, []Metric{
		{
			Count: 2,
			Last:  metric2Value,
			Min:   metric2Value,
			Max:   metricValue,
			Sum:   metricValue + metric2Value,
		},
		{
			Count: 2,
			Last:  metric2Value2,
			Min:   metric2Value2,
			Max:   metricValue2,
			Sum:   metricValue2 + metric2Value2,
		},
	}, metrics)

	assert.Equal(t, 2, sample.HitCount())
}
