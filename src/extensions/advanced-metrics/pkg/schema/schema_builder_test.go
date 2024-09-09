/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package schema

import (
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaBuilder(t *testing.T) {
	builder := NewSchemaBuilder()
	assert.NotNil(t, builder, "SchemaBuilder should not be nil")
	assert.Equal(t, 0, len(builder.fields), "SchemaBuilder should initialize with an empty fields slice")
}

func TestSchemaBuilder_NewDimension(t *testing.T) {
	builder := NewSchemaBuilder()
	builder.NewDimension("dimension1", 10)

	assert.Equal(t, 1, len(builder.fields), "SchemaBuilder should contain one field after adding a dimension")
	assert.Equal(t, "dimension1", builder.fields[0].Name, "The name of the dimension should be set correctly")
}

func TestSchemaBuilder_NewIntegerDimension(t *testing.T) {
	builder := NewSchemaBuilder()
	builder.NewIntegerDimension("intDimension", 100)

	assert.Equal(t, 1, len(builder.fields), "SchemaBuilder should contain one field after adding an integer dimension")
	assert.Equal(t, "intDimension", builder.fields[0].Name, "The name of the integer dimension should be set correctly")

	transformFunc := builder.fields[0].Transform
	assert.NotNil(t, transformFunc, "Integer dimension should have a transform function set")
	assert.Equal(t, &integerDimensionTransformFunction, transformFunc, "Transform function should be correctly set for integer dimension")
}

func TestSchemaBuilder_NewMetric(t *testing.T) {
	builder := NewSchemaBuilder()
	builder.NewMetric("metric1")

	assert.Equal(t, 1, len(builder.fields), "SchemaBuilder should contain one field after adding a metric")
	assert.Equal(t, "metric1", builder.fields[0].Name, "The name of the metric should be set correctly")
}

func TestSchemaBuilder_Build_Success(t *testing.T) {
	builder := NewSchemaBuilder()
	builder.NewDimension("dimension1", 10, WithCollapsingLevel(50))
	builder.NewMetric("metric1")

	sch, err := builder.Build()
	require.NoError(t, err, "Build should succeed with valid configuration")
	assert.NotNil(t, sch, "Schema should not be nil after build")
	assert.Equal(t, 1, len(sch.Metrics()), "Schema should have one metric")
	assert.Equal(t, 1, len(sch.Dimensions()), "Schema should have one dimension")
}

func TestSchemaBuilder_Build_Failure_CollapsingLevel(t *testing.T) {
	builder := NewSchemaBuilder()
	invalidLevel := limits.MaxCollapseLevel + 1
	builder.NewDimension("dimension1", 10, WithCollapsingLevel(invalidLevel))
	builder.NewMetric("metric1")

	sch, err := builder.Build()
	assert.Error(t, err, "Build should fail if a dimension has a collapsing level greater than the maximum allowed")
	assert.Nil(t, sch, "Schema should be nil if build fails")
	assert.Contains(t, err.Error(), "greater than maximum allowed value", "Error message should indicate invalid collapsing level")
}

func TestIntegerDimensionTransformFromData(t *testing.T) {
	data := []byte("1a")
	expectedValue := 26
	value, err := integerDimensionTransformFromData(data)
	require.NoError(t, err, "integerDimensionTransformFromData should succeed for valid input")
	assert.Equal(t, expectedValue, value, "Transform function should correctly parse hex string to int")

	invalidData := []byte("zz")
	_, err = integerDimensionTransformFromData(invalidData)
	assert.Error(t, err, "integerDimensionTransformFromData should fail for invalid hex string")
}

func TestIntegerDimensionTransformFromLookupCode(t *testing.T) {
	code := 42
	expectedString := "42"
	str, err := integerDimensionTransformFromLookupCode(code)
	require.NoError(t, err, "integerDimensionTransformFromLookupCode should succeed for valid input")
	assert.Equal(t, expectedString, str, "Transform function should correctly convert int to string")
}
