/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Float64Gauge struct {
	observations map[attribute.Set]float64
}

func NewFloat64Gauge() *Float64Gauge {
	return &Float64Gauge{observations: make(map[attribute.Set]float64)}
}

func (f *Float64Gauge) Get() map[attribute.Set]float64 {
	return f.observations
}

func (f *Float64Gauge) Set(val float64, attrs attribute.Set) {
	f.observations[attrs] = val
}

func (f *Float64Gauge) Delete(attrs attribute.Set) {
	delete(f.observations, attrs)
}

func (f *Float64Gauge) Callback(ctx context.Context, o metric.Float64Observer) error {
	for attrs, val := range f.observations {
		o.Observe(val, metric.WithAttributeSet(attrs))
	}
	return nil
}

type Int64Gauge struct {
	observations map[attribute.Set]int64
}

func NewInt64Gauge() *Int64Gauge {
	return &Int64Gauge{observations: make(map[attribute.Set]int64)}
}

func (ig *Int64Gauge) Get() map[attribute.Set]int64 {
	return ig.observations
}

func (ig *Int64Gauge) Set(val int64, attrs attribute.Set) {
	ig.observations[attrs] = val
}

func (ig *Int64Gauge) Delete(attrs attribute.Set) {
	delete(ig.observations, attrs)
}

func (ig *Int64Gauge) Callback(ctx context.Context, o metric.Int64Observer) error {
	for attrs, val := range ig.observations {
		o.Observe(val, metric.WithAttributeSet(attrs))
	}
	return nil
}
