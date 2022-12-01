/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package tables

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/mocks"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/stretchr/testify/assert"
)

const (
	dim1LookupCode = 1
	dim2LookupCode = 2

	dim2TransformValue = 142
)

var (
	metric1Value = uint64(0x11)
	metric2Value = uint64(0x12)
	dim1Val      = []byte("dim1Val")
	dim2Val      = []byte("dim2Val")
	dim1ValRaw   = []byte("\"dim1Val\"")
	dim2ValRaw   = []byte("\"dim2Val\"")

	dim1KeyCard = uint32(0xff)
	dim2KeyCard = uint32(0xfff)
	dim1KeyBits = 8
	dim2KeyBits = 12
)

func TestStagingTableAdd(t *testing.T) {

	type dimDef struct {
		dimValue      []byte
		dimLookupCode int
	}
	tests := []struct {
		name             string
		schemaFiels      []*schema.Field
		lookupDimensions []dimDef
		data             [][]byte
		metricsValues    []*uint64
		expectedSamples  []sample.Sample
		shouldFail       bool
	}{
		{
			name: "all dimensions and metrics",
			schemaFiels: []*schema.Field{
				schema.NewDimensionField("dim1", dim1KeyCard),
				schema.NewMetricField("m1"),
				schema.NewDimensionField("dim2", dim2KeyCard),
				schema.NewMetricField("m2"),
			},
			lookupDimensions: []dimDef{
				{
					dimValue:      dim1Val,
					dimLookupCode: dim1LookupCode,
				},
				{
					dimValue:      dim2Val,
					dimLookupCode: dim2LookupCode,
				},
			},
			metricsValues: []*uint64{
				&metric1Value,
				&metric2Value,
			},
			data: [][]byte{
				dim1ValRaw,
				[]byte(fmt.Sprintf("%x", metric1Value)),
				dim2ValRaw,
				[]byte(fmt.Sprintf("%x", metric2Value)),
			},
			expectedSamples: []sample.Sample{
				testSample(t,
					dim1KeyBits+dim2KeyBits,
					[]keyPart{
						{dim1KeyBits, dim1LookupCode},
						{dim2KeyBits, dim2LookupCode},
					},
					2, metric1Value, metric2Value),
			},
		},
		{
			name: "dimension with transform is not stored in lookup, transform result is used as code",
			schemaFiels: []*schema.Field{
				schema.NewDimensionField("dim1", dim1KeyCard),
				schema.NewMetricField("m1"),
				schema.NewDimensionField("dim2", dim2KeyCard,
					schema.WithTransformFunction(&schema.DimensionTransformFunction{FromDataToLookupCode: func(b []byte) (int, error) { return dim2TransformValue, nil }})),
				schema.NewMetricField("m2"),
			},
			lookupDimensions: []dimDef{
				{
					dimValue:      dim1Val,
					dimLookupCode: dim1LookupCode,
				},
			},
			metricsValues: []*uint64{
				&metric1Value,
				&metric2Value,
			},
			data: [][]byte{
				dim1ValRaw,
				[]byte(fmt.Sprintf("%x", metric1Value)),
				dim2ValRaw,
				[]byte(fmt.Sprintf("%x", metric2Value)),
			},
			expectedSamples: []sample.Sample{
				testSample(t,
					dim1KeyBits+dim2KeyBits,
					[]keyPart{
						{dim1KeyBits, dim1LookupCode},
						{dim2KeyBits, dim2TransformValue},
					},
					2, metric1Value, metric2Value),
			},
		},
		{
			name: "empty dimension will be not stored in lookup table",
			schemaFiels: []*schema.Field{
				schema.NewDimensionField("dim1", dim1KeyCard),
				schema.NewMetricField("m1"),
				schema.NewDimensionField("dim2", dim2KeyCard),
				schema.NewMetricField("m2"),
			},
			lookupDimensions: []dimDef{
				{
					dimValue:      dim1Val,
					dimLookupCode: dim1LookupCode,
				},
			},
			metricsValues: []*uint64{
				&metric1Value,
				&metric2Value,
			},
			data: [][]byte{
				dim1ValRaw,
				[]byte(fmt.Sprintf("%x", metric1Value)),
				nil,
				[]byte(fmt.Sprintf("%x", metric2Value)),
			},
			expectedSamples: []sample.Sample{
				testSample(t,
					dim1KeyBits+dim2KeyBits,
					[]keyPart{
						{dim1KeyBits, dim1LookupCode},
						{dim2KeyBits, lookup.LookupNACode},
					},
					2, metric1Value, metric2Value),
			},
		},
		{
			name: "empty metric will be not stored in metrics",
			schemaFiels: []*schema.Field{
				schema.NewDimensionField("dim1", dim1KeyCard),
				schema.NewMetricField("m1"),
				schema.NewDimensionField("dim2", dim2KeyCard),
				schema.NewMetricField("m2"),
			},
			lookupDimensions: []dimDef{
				{
					dimValue:      dim1Val,
					dimLookupCode: dim1LookupCode,
				},
				{
					dimValue:      dim2Val,
					dimLookupCode: dim2LookupCode,
				},
			},
			metricsValues: []*uint64{
				&metric1Value,
				&metric2Value,
			},
			data: [][]byte{
				dim1ValRaw,
				[]byte(fmt.Sprintf("%x", metric1Value)),
				dim2ValRaw,
				nil,
			},
			expectedSamples: []sample.Sample{
				testSample(t,
					dim1KeyBits+dim2KeyBits,
					[]keyPart{
						{dim1KeyBits, dim1LookupCode},
						{dim2KeyBits, dim2LookupCode},
					},
					2, metric1Value),
			},
		},

		{
			name: "lower number of fields than in schema should result in error",
			schemaFiels: []*schema.Field{
				schema.NewDimensionField("dim1", dim1KeyCard),
				schema.NewMetricField("m1"),
				schema.NewDimensionField("dim2", dim2KeyCard),
				schema.NewMetricField("m2"),
			},
			lookupDimensions: []dimDef{
				{
					dimValue:      dim1Val,
					dimLookupCode: dim1LookupCode,
				},
			},
			metricsValues: []*uint64{
				&metric1Value,
			},
			data: [][]byte{
				dim1ValRaw,
				[]byte(fmt.Sprintf("%x", metric1Value)),
			},
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			s := schema.NewSchema(test.schemaFiels...)

			readSamples := mocks.NewMockSamples(ctrl)
			writeSamples := mocks.NewMockSamples(ctrl)
			lookupSet := mocks.NewMockLookupSet(ctrl)

			for dimIndex, dimDef := range test.lookupDimensions {
				lookupSet.EXPECT().
					LookupBytes(dimIndex, dimDef.dimValue).
					Return(dimDef.dimLookupCode, nil)
			}

			for _, expectedSample := range test.expectedSamples {
				writeSamples.EXPECT().Add(expectedSample)
			}
			emptyTableSize := 0
			writeSamples.EXPECT().Len().Return(emptyTableSize).AnyTimes()

			fieldIterator := mocks.NewFieldIteratorStub(test.data)
			stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet { return lookupSet }, testLimit(t))
			err := stagingTables.Add(fieldIterator)
			assert.Equal(t, test.shouldFail, err != nil)
		})
	}
}

func TestStagingTableAddFailOnLookupFail(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := schema.NewSchema(schema.NewDimensionField("dim1", dim1KeyCard))

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)
	testLookupError := errors.New("lookup error")

	emptyTableSize := 0
	writeSamples.EXPECT().Len().Return(emptyTableSize).AnyTimes()

	lookupSet.EXPECT().
		LookupBytes(gomock.Any(), gomock.Any()).
		Return(0, testLookupError)

	fieldIterator := mocks.NewFieldIteratorStub([][]byte{[]byte("dummy")})
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet { return lookupSet }, testLimit(t))
	err := stagingTables.Add(fieldIterator)
	assert.ErrorIs(t, err, testLookupError)
}

func TestStagingTableAddFailOnMetricParseError(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := schema.NewSchema(schema.NewMetricField("metric"))

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)

	emptyTableSize := 0
	writeSamples.EXPECT().Len().Return(emptyTableSize).AnyTimes()

	fieldIterator := mocks.NewFieldIteratorStub([][]byte{[]byte("wrongMetricNotHexInt")})
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet { return lookupSet }, testLimit(t))
	err := stagingTables.Add(fieldIterator)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
}

func TestStagingTableReadSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := schema.NewSchema(schema.NewMetricField("metric"))

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)

	emptyTableSize := 0
	writeSamples.EXPECT().Len().Return(emptyTableSize).AnyTimes()

	newLookupSetCallCount := 0
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet {
		newLookupSetCallCount++
		return lookupSet
	}, testLimit(t))

	readSamples.EXPECT().Clear()

	readTable, lookupSnapshot := stagingTables.ReadSnapshot(false)

	assert.Equal(t, writeSamples, readTable)
	assert.Equal(t, nil, lookupSnapshot)
	assert.Equal(t, 1, newLookupSetCallCount)
}

func TestStagingTableReadSnapshotWithLookupReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := schema.NewSchema(schema.NewMetricField("metric"))

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)

	emptyTableSize := 0
	writeSamples.EXPECT().Len().Return(emptyTableSize).AnyTimes()

	newLookupSetCallCount := 0
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet {
		newLookupSetCallCount++
		return lookupSet
	}, testLimit(t))

	readSamples.EXPECT().Clear()

	readTable, lookupSnapshot := stagingTables.ReadSnapshot(true)

	assert.Equal(t, writeSamples, readTable)
	assert.Equal(t, lookupSet, lookupSnapshot)
	assert.Equal(t, 2, newLookupSetCallCount)

	writeSamples.EXPECT().Clear()

	readTable, lookupSnapshot = stagingTables.ReadSnapshot(false)

	assert.Equal(t, readSamples, readTable)
	assert.Equal(t, nil, lookupSnapshot)
	assert.Equal(t, 2, newLookupSetCallCount)
}

func TestStagingTableWithDimensionsCollapsing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const max = 100
	const threshold = 90
	limit, err := limits.NewLimits(max, threshold)
	assert.NoError(t, err)
	currentSize := threshold + 1

	data := [][]byte{
		dim1ValRaw,
		dim2ValRaw,
		[]byte(fmt.Sprintf("%x", metric1Value)),
	}

	const collapsedDimensionLevel = 1
	dim1 := schema.NewDimensionField("dim1", dim1KeyCard, schema.WithLevel(collapsedDimensionLevel))
	const nonCollapsedDimensionLevel = 99
	dim2 := schema.NewDimensionField("dim2", dim2KeyCard, schema.WithLevel(nonCollapsedDimensionLevel))

	expectedSample := testSample(t,
		dim1.KeyBitSize+dim2.KeyBitSize,
		[]keyPart{
			{dim1.KeyBitSize, lookup.LookupAggrCode},
			{dim2.KeyBitSize, dim2LookupCode},
		},
		1, metric1Value)

	s := schema.NewSchema([]*schema.Field{
		dim1,
		dim2,
		schema.NewMetricField("m1"),
	}...)

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)

	lookupSet.EXPECT().
		LookupBytes(dim2.Index(), dim2Val).
		Return(dim2LookupCode, nil)

	writeSamples.EXPECT().Add(expectedSample)
	writeSamples.EXPECT().Len().Return(currentSize).AnyTimes()

	fieldIterator := mocks.NewFieldIteratorStub(data)
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet { return lookupSet }, limit)

	err = stagingTables.Add(fieldIterator)
	assert.NoError(t, err)
}

func TestStagingTableFullSamplesTableShouldCollapseAllDimensions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const max = 100
	const threshold = 90
	limit, err := limits.NewLimits(max, threshold)
	assert.NoError(t, err)
	currentSize := max + 1

	data := [][]byte{
		dim1ValRaw,
		dim2ValRaw,
		[]byte(fmt.Sprintf("%x", metric1Value)),
	}

	const collapsedDimensionLevel = 1
	dim1 := schema.NewDimensionField("dim1", dim1KeyCard, schema.WithLevel(collapsedDimensionLevel))
	dim2 := schema.NewDimensionField("dim2", dim2KeyCard, schema.WithLevel(limits.MaxCollapseLevel-1))

	expectedSample := testSample(t,
		dim1.KeyBitSize+dim2.KeyBitSize,
		[]keyPart{
			{dim1.KeyBitSize, lookup.LookupAggrCode},
			{dim2.KeyBitSize, lookup.LookupAggrCode},
		},
		1, metric1Value)

	s := schema.NewSchema([]*schema.Field{
		dim1,
		dim2,
		schema.NewMetricField("m1"),
	}...)

	readSamples := mocks.NewMockSamples(ctrl)
	writeSamples := mocks.NewMockSamples(ctrl)
	lookupSet := mocks.NewMockLookupSet(ctrl)

	writeSamples.EXPECT().Add(expectedSample)
	writeSamples.EXPECT().Len().Return(currentSize).AnyTimes()

	fieldIterator := mocks.NewFieldIteratorStub(data)
	stagingTables := newStagingTable(s, readSamples, writeSamples, func(*schema.Schema) LookupSet { return lookupSet }, limit)

	err = stagingTables.Add(fieldIterator)
	assert.NoError(t, err)
}

func TestStagingTableThreadSafety(t *testing.T) {
	mu := &sync.Mutex{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := schema.NewSchema(
		schema.NewDimensionField("dim1", dim1KeyCard),
		schema.NewMetricField("m1"),
	)

	testData := [][]byte{
		dim1ValRaw,
		[]byte(fmt.Sprintf("%x", metric1Value)),
	}

	stagingTables := NewStagingTable(s, testLimit(t))

	const repetition = 100
	wg := sync.WaitGroup{}
	wg.Add(repetition)
	for i := 0; i < repetition; i++ {
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			fieldIterator := mocks.NewFieldIteratorStub(testData)
			err := stagingTables.Add(fieldIterator)
			assert.NoError(t, err)
		}()
	}

	var resultSample *sample.Sample

	for i := 0; i < 15; i++ {
		samplesView, lookupSet := stagingTables.ReadSnapshot(false)
		assert.Equal(t, nil, lookupSet)
		singleSample := readSingleSample(t, samplesView)
		if resultSample == nil {
			resultSample = singleSample
		} else if singleSample != nil {
			err := resultSample.AddSample(singleSample)
			assert.NoError(t, err)
		}
	}
	wg.Wait()

	samplesView, lookupSet := stagingTables.ReadSnapshot(true)
	assert.NotNil(t, lookupSet)
	singleSample := readSingleSample(t, samplesView)
	if resultSample == nil {
		resultSample = singleSample
	} else if singleSample != nil {
		err := resultSample.AddSample(singleSample)
		assert.NoError(t, err)
	}

	const lookupCode = 2
	expectedSample := testSample(t,
		dim1KeyBits,
		[]keyPart{
			{dim1KeyBits, lookupCode},
		},
		1, metric1Value)
	expectedSample.Metrics()[0] = sample.Metric{
		Count: repetition,
		Last:  float64(metric1Value),
		Min:   float64(metric1Value),
		Max:   float64(metric1Value),
		Sum:   float64(metric1Value * repetition),
	}
	expectedSample.AddHitCount(repetition - 1)

	data, err := lookupSet.LookupCode(s.Dimension(0).Index(), lookupCode)
	assert.NoError(t, err)
	assert.Equal(t, string(dim1Val), data)
	assert.Equal(t, expectedSample, *resultSample)
}

func TestTrimDimensionValue(t *testing.T) {
	assert.Nil(t, trimDimensionValue(nil))
	assert.Equal(t, []byte(""), trimDimensionValue([]byte("")))
	assert.Equal(t, []byte("a"), trimDimensionValue([]byte("a")))
	assert.Equal(t, []byte("ab"), trimDimensionValue([]byte("ab")))
	assert.Equal(t, []byte("abc"), trimDimensionValue([]byte("abc")))
	assert.Equal(t, []byte("abc"), trimDimensionValue([]byte("\"abc")))
	assert.Equal(t, []byte("abc"), trimDimensionValue([]byte("abc\"")))
	assert.Equal(t, []byte("abc"), trimDimensionValue([]byte("\"abc\"")))
	assert.Equal(t, []byte("a\"bc"), trimDimensionValue([]byte("a\"bc")))
}

func readSingleSample(t *testing.T, samples SamplesView) *sample.Sample {
	result := []sample.Sample{}
	samples.Range(func(sample *sample.Sample) {
		result = append(result, *sample)
	})

	if len(result) == 0 {
		return nil
	}
	assert.Equal(t, 1, len(result))
	return &result[0]
}

type keyPart struct {
	bits int
	part int
}

func testSample(t *testing.T, keyBitSize int, keyParts []keyPart, metricsCount int, metrics ...uint64) sample.Sample {
	sample := sample.NewSample(keyBitSize, metricsCount)
	for i, v := range metrics {
		err := sample.SetMetric(i, float64(v))
		assert.NoError(t, err)
	}

	for _, k := range keyParts {
		err := sample.Key().AddKeyPart(k.part, k.bits)
		assert.NoError(t, err)
	}

	return sample
}

func testLimit(t *testing.T) limits.Limits {
	const max = 100
	const threshold = 90
	limit, err := limits.NewLimits(max, threshold)
	assert.NoError(t, err)
	return limit
}
