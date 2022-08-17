package tables

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	log "github.com/sirupsen/logrus"
)

//go:generate mockgen -source staging_table.go -destination mocks/staging_table_mocks.go -package mocks
const (
	hexBase = 16
)

type FieldIterator interface {
	Next() []byte
	HasNext() bool
}

type SamplesView interface {
	Range(cb func(sample *sample.Sample))
}

type Samples interface {
	SamplesView
	Add(s sample.Sample) error
	Clear()
	Len() int
}

type LookupSet interface {
	LookupBytes(schema.FieldIndex, []byte) (int, error)
	LookupCode(int, int) (string, error)
}

// StagingTable is a table structure responsible for storing samples with metrics and dimensions.
// StagingTable uses two Samples tables one for writes and one for reads in order to prevent lock contention.
type StagingTable struct {
	sync.RWMutex

	lookups LookupSet
	write   Samples
	read    Samples
	schema  *schema.Schema

	newLookupSetFunc newLookupSetFunc

	limits limits.Limits
}

func NewStagingTable(s *schema.Schema, limits limits.Limits) *StagingTable {
	return newStagingTable(s, sample.NewSampleTable(), sample.NewSampleTable(), newLookupSet, limits)
}

func newStagingTable(s *schema.Schema, readSamples, writeSamples Samples, newLookupSet newLookupSetFunc, limits limits.Limits) *StagingTable {
	return &StagingTable{
		lookups:          newLookupSet(s),
		write:            writeSamples,
		read:             readSamples,
		schema:           s,
		newLookupSetFunc: newLookupSet,
		limits:           limits,
	}
}

type newLookupSetFunc = func(*schema.Schema) LookupSet

func newLookupSet(s *schema.Schema) LookupSet {
	return lookup.NewLookupSetFromSchema(s.Dimensions())
}

// Add adds input sample in form of fields to write table.
// Single field could be a metric or a dimension, interpretation depend on the defined schema.
// Fields are interpreted in the order of fields defined in the table schema.
// Fields format:
// - metric: number represented as hex string
// - dimension: string representing dimension value, or in format which will be used by Field transform function
func (t *StagingTable) Add(fieldIterator FieldIterator) error {
	t.RLock()
	defer t.RUnlock()

	sample := sample.NewSample(t.schema.KeySize(), t.schema.NumMetrics())
	sampleKey := sample.Key()

	currentCollapsingLevel := t.limits.GetCurrentCollapsingLevel(t.write.Len())

	for _, field := range t.schema.Fields() {
		if !fieldIterator.HasNext() {
			return fmt.Errorf("number of fields differ from schema definition")
		}
		fieldData := fieldIterator.Next()

		switch field.Type {
		case schema.FieldTypeDimension:
			dimensionLookupCode := lookup.LookupAggrCode
			if !field.ShouldCollapse(currentCollapsingLevel) {
				code, err := t.lookupCodeForDimension(trimDimensionValue(fieldData), field)
				if err != nil {
					return fmt.Errorf("cannot lookup %s field '%v' data: %w", field.Name, fieldData, err)
				}
				dimensionLookupCode = code
			}
			err := sampleKey.AddKeyPart(dimensionLookupCode, field.KeyBitSize)
			if err != nil {
				return fmt.Errorf("setting key %d bits in field %s: %w", dimensionLookupCode, field.Name, err)
			}
		case schema.FieldTypeMetric:
			if len(fieldData) == 0 {
				continue
			}
			val, err := strconv.ParseUint(string(fieldData), hexBase, 64)
			if err != nil {
				return fmt.Errorf("cannot parse '%s' metrics '%v' value: %w", field.Name, fieldData, err)
			}

			err = sample.SetMetric(field.Index(), float64(val))
			if err != nil {
				return fmt.Errorf("cannot store '%v' metric: %w", field.Name, err)
			}
		}
	}

	if fieldIterator.HasNext() {
		log.Warning("Received message has more fields that specified schema.")
	}

	err := t.write.Add(sample)
	return err
}

func (t *StagingTable) lookupCodeForDimension(dimension []byte, field *schema.Field) (int, error) {
	switch {
	case len(dimension) == 0:
		return lookup.LookupNACode, nil
	case field.Transform != nil:
		return field.Transform.FromDataToLookupCode(dimension)
	default:
		return t.lookups.LookupBytes(field.Index(), dimension)
	}
}

// ReadSnapshot creates samples read snapshot and returns view on it.
// Previous read snapshot will be cleared.
// By default returned LookupSet is nil.
// resetLookups determines if new instance of LookupSet should be created, old one will be returned in case of reset
//				,as Samples and LookupSet reset need to be done atomically this function
// 				do both functionalities instead of separate methods
func (t *StagingTable) ReadSnapshot(resetLookups bool) (SamplesView, LookupSet) {
	t.Lock()
	defer t.Unlock()

	tmp := t.write
	t.write = t.read
	t.read = tmp

	t.write.Clear()

	var lookups LookupSet
	if resetLookups {
		lookups = t.lookups
		t.lookups = t.newLookupSetFunc(t.schema)
	}

	return t.read, lookups
}

// trimDimensionValue remove from dimension value enclosing quotes
func trimDimensionValue(dim []byte) []byte {
	const Quote = "\""
	if len(dim) < 2 {
		return dim
	}
	return []byte(strings.Trim(string(dim), Quote))
}
