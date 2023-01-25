/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package schema

type Schema struct {
	fields     []*Field
	dimensions []*Field
	metrics    []*Field
	keySize    int
	sizes      []int
}

func NewSchema(fields ...*Field) *Schema {
	s := &Schema{
		fields: fields,
	}

	dims := make([]*Field, 0)
	metrics := make([]*Field, 0)
	keyBits := 0

	metricsIndex := 0
	dimensionIndex := 0

	for _, f := range s.fields {
		if f.Type == FieldTypeDimension {
			dims = append(dims, f)

			f.KeyBitPositionInCompoundKey = keyBits
			keyBits += f.KeyBitSize
			f.index = dimensionIndex
			dimensionIndex++
		}
		if f.Type == FieldTypeMetric {
			metrics = append(metrics, f)

			f.index = metricsIndex
			metricsIndex++
		}
	}

	s.dimensions = dims
	s.metrics = metrics
	s.keySize = keyBits

	s.sizes = make([]int, 0)
	for _, d := range s.dimensions {
		s.sizes = append(s.sizes, d.KeyBitSize)
	}

	return s
}

func (s *Schema) Field(i int) *Field {
	return s.fields[i]
}

func (s *Schema) Fields() []*Field {
	return s.fields
}

func (s *Schema) Metric(i int) *Field {
	return s.metrics[i]
}

func (s *Schema) Metrics() []*Field {
	return s.metrics
}

func (s *Schema) Dimension(i int) *Field {
	return s.dimensions[i]
}

func (s *Schema) Dimensions() []*Field {
	return s.dimensions
}

func (s *Schema) NumMetrics() int {
	return len(s.metrics)
}

func (s *Schema) NumDimensions() int {
	return len(s.dimensions)
}

func (s *Schema) KeySize() int {
	return s.keySize
}

func (s *Schema) DimensionKeyPartSizes() []int {
	return s.sizes
}
