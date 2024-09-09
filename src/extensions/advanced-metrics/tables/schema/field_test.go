/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package schema

import (
	"math"
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/stretchr/testify/assert"
)

func TestNewDimensionField(t *testing.T) {
	name := "dimension1"
	maxDimensionSetSize := uint32(16)

	// Create a new dimension field
	field := NewDimensionField(name, maxDimensionSetSize)

	assert.NotNil(t, field, "NewDimensionField should not return nil")
	assert.Equal(t, name, field.Name, "Field name should be set correctly")
	assert.Equal(t, FieldTypeDimension, field.Type, "Field type should be set to FieldTypeDimension")
	assert.Equal(t, maxDimensionSetSize, field.MaxDimensionSetSize, "MaxDimensionSetSize should be set correctly")
	assert.Equal(t, int(math.Log2(float64(maxDimensionSetSize)))+1, field.KeyBitSize, "KeyBitSize should be calculated correctly")
	assert.Nil(t, field.CollapsingLevel, "CollapsingLevel should be nil by default")
}

func TestNewDimensionField_WithOptions(t *testing.T) {
	name := "dimensionWithOptions"
	maxDimensionSetSize := uint32(16)
	customKeyBitSize := 8
	collapsingLevel := limits.CollapsingLevel(50)
	transformFunc := &DimensionTransformFunction{
		FromDataToLookupCode:  func(data []byte) (int, error) { return 1, nil },
		FromLookupCodeToValue: func(code int) (string, error) { return "value", nil },
	}

	// Create a new dimension field with options
	field := NewDimensionField(name, maxDimensionSetSize,
		WithKeyBitSize(customKeyBitSize),
		WithLevel(collapsingLevel),
		WithTransformFunction(transformFunc),
	)

	assert.Equal(t, customKeyBitSize, field.KeyBitSize, "KeyBitSize should be set by WithKeyBitSize option")
	assert.Equal(t, &collapsingLevel, field.CollapsingLevel, "CollapsingLevel should be set by WithLevel option")
	assert.Equal(t, transformFunc, field.Transform, "Transform function should be set by WithTransformFunction option")
}

func TestNewMetricField(t *testing.T) {
	name := "metric1"

	// Create a new metric field
	field := NewMetricField(name)

	assert.NotNil(t, field, "NewMetricField should not return nil")
	assert.Equal(t, name, field.Name, "Field name should be set correctly")
	assert.Equal(t, FieldTypeMetric, field.Type, "Field type should be set to FieldTypeMetric")
	assert.Equal(t, uint32(0), field.MaxDimensionSetSize, "MaxDimensionSetSize should be 0 for metric fields")
	assert.Equal(t, 0, field.KeyBitSize, "KeyBitSize should be 0 for metric fields")
	assert.Nil(t, field.CollapsingLevel, "CollapsingLevel should be nil for metric fields")
	assert.Nil(t, field.Transform, "Transform function should be nil for metric fields")
}

func TestField_Index(t *testing.T) {
	field := NewMetricField("metric1")
	field.index = 5

	assert.Equal(t, 5, field.Index(), "Field index should return the correct value")
}

func TestField_ShouldCollapse(t *testing.T) {
	field := NewDimensionField("dimension1", 16)
	level := limits.CollapsingLevel(50)

	// Test when CollapsingLevel is nil
	assert.False(t, field.ShouldCollapse(level), "ShouldCollapse should return false if CollapsingLevel is nil")

	// Test when CollapsingLevel is set
	field.CollapsingLevel = &level
	assert.False(t, field.ShouldCollapse(level), "ShouldCollapse should return false if level is equal to CollapsingLevel")

	// Test when level is greater than CollapsingLevel
	assert.True(t, field.ShouldCollapse(level+1), "ShouldCollapse should return true if level is greater than CollapsingLevel")

	// Test when level is less than CollapsingLevel
	assert.False(t, field.ShouldCollapse(level-1), "ShouldCollapse should return false if level is less than CollapsingLevel")
}
