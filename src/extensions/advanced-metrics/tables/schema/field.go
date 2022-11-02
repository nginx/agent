package schema

import (
	"math"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
)

type FieldType int

const (
	FieldTypeDimension FieldType = iota
	FieldTypeMetric
)

type FieldIndex = int

// Field defines attributes of input field for StagingTable
type Field struct {
	Name string
	Type FieldType

	DimensionField

	index FieldIndex
}

// DimensionField defines dimension field specific information
// KeyBitSize specifies the size of the key which will be used in the staging table to represent dimension value
// MaxDimensionSetSize specifies max unique dimension which will be stored in staging table
//
//	if max size will be reaches all new unique dimensions will be transformed to AGGR value
type DimensionField struct {
	KeyBitSize                  int
	KeyBitPositionInCompoundKey int
	MaxDimensionSetSize         uint32
	Transform                   *DimensionTransformFunction
	CollapsingLevel             *limits.CollapsingLevel
}

type FieldOption func(f *Field)

type DimensionTransformFunction struct {
	FromDataToLookupCode  func([]byte) (int, error)
	FromLookupCodeToValue func(int) (string, error)
}

func WithTransformFunction(t *DimensionTransformFunction) FieldOption {
	return func(f *Field) { f.Transform = t }
}

func WithKeyBitSize(keyBitSize int) FieldOption {
	return func(f *Field) { f.KeyBitSize = keyBitSize }
}

func WithLevel(level limits.CollapsingLevel) FieldOption {
	return func(f *Field) { f.CollapsingLevel = &level }
}

func NewDimensionField(name string, maxDimensionSetSize uint32, opts ...FieldOption) *Field {
	f := &Field{
		Name: name,
		Type: FieldTypeDimension,
		DimensionField: DimensionField{
			MaxDimensionSetSize: maxDimensionSetSize,
			KeyBitSize:          int(math.Log2(float64(maxDimensionSetSize))) + 1,
			CollapsingLevel:     nil,
		},
	}

	for _, o := range opts {
		o(f)
	}

	return f
}

func NewMetricField(name string) *Field {
	return &Field{
		Name: name,
		Type: FieldTypeMetric,
	}
}

func (f *Field) Index() FieldIndex {
	return f.index
}

func (f *Field) ShouldCollapse(level limits.CollapsingLevel) bool {
	return f.CollapsingLevel != nil && level > *f.CollapsingLevel
}
