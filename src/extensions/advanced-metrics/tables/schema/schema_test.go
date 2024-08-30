/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchema(t *testing.T) {
	field1 := &Field{
		Name: "dimension1",
		Type: FieldTypeDimension,
		DimensionField: DimensionField{
			KeyBitSize:                  8,
			KeyBitPositionInCompoundKey: 0,
			MaxDimensionSetSize:         0,
			Transform:                   &DimensionTransformFunction{},
			CollapsingLevel:             new(uint32),
		},
	}
	field2 := &Field{
		Name: "metric1",
		Type: FieldTypeMetric,
		DimensionField: DimensionField{
			KeyBitSize:                  0,
			KeyBitPositionInCompoundKey: 0,
			MaxDimensionSetSize:         0,
			Transform:                   &DimensionTransformFunction{},
			CollapsingLevel:             new(uint32),
		},
	}
	field3 := &Field{
		Name: "dimension2",
		Type: FieldTypeDimension,
		DimensionField: DimensionField{
			KeyBitSize:                  16,
			KeyBitPositionInCompoundKey: 0,
			MaxDimensionSetSize:         0,
			Transform:                   &DimensionTransformFunction{},
			CollapsingLevel:             new(uint32),
		},
	}

	schema := NewSchema(field1, field2, field3)

	// Test Schema initialization
	assert.NotNil(t, schema, "NewSchema should not return nil")
	assert.Equal(t, 3, len(schema.Fields()), "Schema should have 3 fields")
	assert.Equal(t, 2, schema.NumDimensions(), "Schema should have 2 dimensions")
	assert.Equal(t, 1, schema.NumMetrics(), "Schema should have 1 metric")
	assert.Equal(t, 24, schema.KeySize(), "Key size should be the sum of KeyBitSizes for dimensions (8 + 16)")

	// Test Field indexing and KeyBitPositionInCompoundKey
	assert.Equal(t, 0, field1.KeyBitPositionInCompoundKey, "Field1 KeyBitPositionInCompoundKey should be 0")
	assert.Equal(t, 8, field3.KeyBitPositionInCompoundKey, "Field3 KeyBitPositionInCompoundKey should be 8")
	assert.Equal(t, 0, field1.index, "Field1 index should be 0")
	assert.Equal(t, 1, field3.index, "Field3 index should be 1")
	assert.Equal(t, 0, field2.index, "Field2 index should be 0")

	// Test DimensionKeyPartSizes
	assert.Equal(t, []int{8, 16}, schema.DimensionKeyPartSizes(), "DimensionKeyPartSizes should return correct bit sizes")
}

func TestSchema_Field(t *testing.T) {
	field1 := &Field{Name: "dimension1"}
	field2 := &Field{Name: "metric1"}

	schema := NewSchema(field1, field2)

	assert.Equal(t, field1, schema.Field(0), "Field(0) should return the correct field")
	assert.Equal(t, field2, schema.Field(1), "Field(1) should return the correct field")
}

func TestSchema_Metric(t *testing.T) {
	field := &Field{Name: "metric1", Type: FieldTypeMetric}
	schema := NewSchema(field)

	assert.Equal(t, field, schema.Metric(0), "Metric(0) should return the correct metric field")
}

func TestSchema_Dimension(t *testing.T) {
	field := &Field{Name: "dimension1", Type: FieldTypeDimension}
	schema := NewSchema(field)

	assert.Equal(t, field, schema.Dimension(0), "Dimension(0) should return the correct dimension field")
}

func TestSchema_NumMetrics(t *testing.T) {
	field1 := &Field{Name: "metric1", Type: FieldTypeMetric}
	field2 := &Field{Name: "metric2", Type: FieldTypeMetric}
	schema := NewSchema(field1, field2)

	assert.Equal(t, 2, schema.NumMetrics(), "NumMetrics should return the correct number of metrics")
}

func TestSchema_NumDimensions(t *testing.T) {
	field1 := &Field{Name: "dimension1", Type: FieldTypeDimension}
	field2 := &Field{Name: "dimension2", Type: FieldTypeDimension}
	schema := NewSchema(field1, field2)

	assert.Equal(t, 2, schema.NumDimensions(), "NumDimensions should return the correct number of dimensions")
}

func TestSchema_KeySize(t *testing.T) {
	field1 := &Field{Name: "dimension1", Type: FieldTypeDimension, DimensionField: DimensionField{KeyBitSize: 8}}
	field2 := &Field{Name: "dimension2", Type: FieldTypeDimension, DimensionField: DimensionField{KeyBitSize: 16}}
	schema := NewSchema(field1, field2)

	assert.Equal(t, 24, schema.KeySize(), "KeySize should return the sum of KeyBitSizes of all dimensions")
}

func TestSchema_DimensionKeyPartSizes(t *testing.T) {
	field1 := &Field{Name: "dimension1", Type: FieldTypeDimension, DimensionField: DimensionField{KeyBitSize: 8}}
	field2 := &Field{Name: "dimension2", Type: FieldTypeDimension, DimensionField: DimensionField{KeyBitSize: 16}}
	schema := NewSchema(field1, field2)

	assert.Equal(t, []int{8, 16}, schema.DimensionKeyPartSizes(), "DimensionKeyPartSizes should return the correct bit sizes for all dimensions")
}
