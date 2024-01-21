/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate

//counterfeiter:generate -o ./mock_float64_observer.go ./ Float64ObserverInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/metric mock_float64_observer.go | sed -e s\\/metric\\\\.\\/\\/g > mock_float64_observer_fixed.go"
//go:generate mv mock_float64_observer_fixed.go mock_float64_observer.go

// TODO: counterfeiter is unable to properly implement the embedded interface for `Float64Observer`, so this does not work.
// Need to figure out another solution for mocking `Float64Observer`.
// type Float64ObserverInterface interface {
// 	Observe(value float64, options ...metric.ObserveOption)

// 	float64Observer()
// }

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

	// TODO: Need a working mocking solution for testing the Callback() method.
	// mockObserver := &FakeFloat64ObserverInterface{}
	// f64gauge.Callback(context.Background(), mockObserver)

	f64gauge.Delete(testSet)
	assert.Equal(t, f64gauge.observations, make(map[attribute.Set]float64))
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

	// TODO: Need a working mocking solution for testing the Callback() method.
	// i64gauge.Callback(context.Background(), mockObserver)

	i64gauge.Delete(testSet)
	assert.Equal(t, i64gauge.observations, make(map[attribute.Set]int64))
}
