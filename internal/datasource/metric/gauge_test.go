/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
)

type MockFloat64Observer struct {
	embedded.Float64Observer
	mock.Mock
}

func (mfo *MockFloat64Observer) Observe(value float64, options ...metric.ObserveOption) {
	mfo.On("Observe").Return()
	mfo.Called()
}

type MockInt64Observer struct {
	embedded.Int64Observer
	mock.Mock
}

func (mfo *MockInt64Observer) Observe(value int64, options ...metric.ObserveOption) {
	mfo.On("Observe").Return()
	mfo.Called()
}

func TestFloat64Gauge(t *testing.T) {
	f64gauge := NewFloat64Gauge()
	assert.NotNil(t, f64gauge)

	assert.Equal(t, f64gauge.observations, make(map[attribute.Set]float64))

	testObs := make(map[attribute.Set]float64)
	testSet := attribute.NewSet(
		attribute.KeyValue{
			Key:   "test-key",
			Value: attribute.StringValue("test-value"),
		},
	)

	testObs[testSet] = 123.0
	f64gauge.Set(123.0, testSet)

	assert.Equal(t, f64gauge.observations, testObs)
	assert.Equal(t, f64gauge.Get(), testObs)

	mockObserver := new(MockFloat64Observer)
	err := f64gauge.Callback(context.Background(), mockObserver)
	assert.NoError(t, err)

	f64gauge.Delete(testSet)
	assert.Equal(t, f64gauge.observations, make(map[attribute.Set]float64))
	mockObserver.AssertExpectations(t)
}

func TestInt64Gauge(t *testing.T) {
	i64gauge := NewInt64Gauge()
	assert.NotNil(t, i64gauge)

	assert.Equal(t, i64gauge.observations, make(map[attribute.Set]int64))

	testObs := make(map[attribute.Set]int64)
	testSet := attribute.NewSet(
		attribute.KeyValue{
			Key:   "test-key",
			Value: attribute.StringValue("test-value"),
		},
	)

	testObs[testSet] = 123
	i64gauge.Set(123, testSet)

	assert.Equal(t, i64gauge.observations, testObs)
	assert.Equal(t, i64gauge.Get(), testObs)

	mockObserver := &MockInt64Observer{}
	err := i64gauge.Callback(context.Background(), mockObserver)
	assert.NoError(t, err)

	i64gauge.Delete(testSet)
	assert.Equal(t, i64gauge.observations, make(map[attribute.Set]int64))
}
