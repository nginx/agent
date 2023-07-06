/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package schema

import (
	"fmt"
	"math/bits"
	"strconv"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
)

type (
	FieldOption                = schema.FieldOption
	DimensionTransformFunction = schema.DimensionTransformFunction
)

// WithTransformFunction defines pair of function which transform dimension raw value
// from []byte to LookupCode and from LookupCode to string when dimension value will be published
// Presence of this pair of functions assumes that dimension will be not stored in LookupTable
// and converted value will be directly encoded in tables key.
var WithTransformFunction = schema.WithTransformFunction

// WithCollapsingLevel defines CollapsingLevel for a dimension.
// CollapsingLevel determines if specific dimension value should be aggregated into "AGGR" value.
// CollapsingLevel is specified as a percent of elements above threshold value for both staging and priority tables
// in corelation to maximum of elements between maximum and threshold config values for specific table.
// This is percent value and should be number within 0-100 range.
// More about collapsing algorithm in TableSizesLimits doc string.
var WithCollapsingLevel = schema.WithLevel

type SchemaBuilder struct {
	fields []*schema.Field
}

func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		fields: make([]*schema.Field, 0),
	}
}

// NewDimension builds new dimension definition
// name - name of the dimension used later by publisher to build Dimension in MetricSet
// maxDimensionSetSize - cardinality of given dimension, specify maximal size of unique dimensions which will be accumulated
//
//	during publishing period, minimum value is 4, two elements are reserved for internal use
func (b *SchemaBuilder) NewDimension(name string, maxDimensionSetSize uint32, opts ...FieldOption) *SchemaBuilder {
	b.fields = append(b.fields, schema.NewDimensionField(name, maxDimensionSetSize, opts...))

	return b
}

func (b *SchemaBuilder) NewIntegerDimension(name string, maxDimensionValue uint32) *SchemaBuilder {
	b.fields = append(b.fields, schema.NewDimensionField(name,
		uint32(maxDimensionValue),
		schema.WithTransformFunction(&integerDimensionTransformFunction),
		schema.WithKeyBitSize(bits.UintSize),
	))

	return b
}

var integerDimensionTransformFunction = schema.DimensionTransformFunction{
	FromDataToLookupCode:  integerDimensionTransformFromData,
	FromLookupCodeToValue: integerDimensionTransformFromLookupCode,
}

func integerDimensionTransformFromData(data []byte) (int, error) {
	res, err := strconv.ParseInt(string(data), 16, 0)
	return int(res), err
}

func integerDimensionTransformFromLookupCode(code int) (string, error) {
	return strconv.Itoa(code), nil
}

func (b *SchemaBuilder) NewMetric(name string) *SchemaBuilder {
	b.fields = append(b.fields, schema.NewMetricField(name))
	return b
}

func (b *SchemaBuilder) Build() (*schema.Schema, error) {
	schema := schema.NewSchema(b.fields...)
	for _, d := range schema.Dimensions() {
		if d.CollapsingLevel != nil && *d.CollapsingLevel > limits.MaxCollapseLevel {
			return nil, fmt.Errorf("dimension: '%s' contains CollapsingLevel=%d greater than maximum allowed value=%d", d.Name, d.CollapsingLevel, limits.MaxCollapseLevel)
		}
	}
	return schema, nil
}
