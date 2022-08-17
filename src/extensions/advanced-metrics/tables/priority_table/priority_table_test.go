package priority_table

import (
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/lookup"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriorityTable(t *testing.T) {
	sh := &schema.Schema{}
	l, err := limits.NewLimits(100, 90)
	assert.NoError(t, err)
	table := NewPriorityTable(sh, l)
	s := sample.NewSample(1, 1)
	assert.NoError(t, table.Add(&s))
	s2 := sample.NewSample(2, 2)
	assert.NoError(t, table.Add(&s2))
	s3 := sample.NewSample(3, 3)
	assert.NoError(t, table.Add(&s3))
	samples := []sample.Sample{}
	for _, s := range table.Samples() {
		samples = append(samples, *s)
	}

	expectedSamples := []sample.Sample{
		sample.NewSample(1, 1),
		sample.NewSample(2, 2),
		sample.NewSample(3, 3),
	}
	assert.ElementsMatch(t, expectedSamples, samples)
}

func TestPriorityTableWithCollapsing(t *testing.T) {
	const (
		dim1KeyCard = uint32(0xff)
		dim2KeyCard = uint32(0xff)
	)

	dim1 := schema.NewDimensionField("dim1", dim1KeyCard, schema.WithLevel(1))
	dim2 := schema.NewDimensionField("dim2", dim2KeyCard)
	testSchema := schema.NewSchema([]*schema.Field{
		dim1,
		dim2,
		schema.NewMetricField("m1"),
	}...)

	testSingleMetric := []sample.Metric{
		{
			Count: 1,
			Last:  1,
			Min:   1,
			Max:   1,
			Sum:   1,
		},
	}

	testTwoSingleMetricAggregated := []sample.Metric{
		{
			Count: 2,
			Last:  1,
			Min:   1,
			Max:   1,
			Sum:   2,
		},
	}
	threshold := 6
	limit := testLimits(t, 10, threshold)
	topPriority := 10
	topPriorityTestSamplesWithThresholdSize := generateUniqueTestSamples(t, threshold, []int{10, 10}, topPriority, testSingleMetric)
	topPriorityTestSamplesWithSizeLowerThanTreshold := generateUniqueTestSamples(t, 5, []int{10, 10}, 10, testSingleMetric)

	tests := []struct {
		name            string
		inputSamples    []*sample.Sample
		expectedSamples []*sample.Sample
		limits          limits.Limits
	}{
		{
			name: "should collapse single metric exceeding threshold",
			inputSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testSingleMetric, []int{100, 100}, 1),
			),
			expectedSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testSingleMetric, []int{lookup.LookupAggrCode, 100}, 1),
			),
			limits: limit,
		},
		{
			name: "should collapse two metric into one exceeding threshold",
			inputSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testSingleMetric, []int{100, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 100}, 1),
			),
			expectedSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testTwoSingleMetricAggregated, []int{lookup.LookupAggrCode, 100}, 2),
			),
			limits: limit,
		},
		{
			name: "should collapse three metric into two exceeding threshold",
			inputSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testSingleMetric, []int{100, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 101}, 1),
			),
			expectedSamples: append(
				topPriorityTestSamplesWithThresholdSize,
				testSample(t, testTwoSingleMetricAggregated, []int{lookup.LookupAggrCode, 100}, 2),
				testSample(t, testSingleMetric, []int{lookup.LookupAggrCode, 101}, 1),
			),
			limits: limit,
		},
		{
			name: "should collapse two metric into one exceeding threshold, and not collapse metric with higher priority",
			inputSamples: append(
				topPriorityTestSamplesWithSizeLowerThanTreshold,
				testSample(t, testSingleMetric, []int{100, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 101}, 3),
			),
			expectedSamples: append(
				topPriorityTestSamplesWithSizeLowerThanTreshold,
				testSample(t, testTwoSingleMetricAggregated, []int{lookup.LookupAggrCode, 100}, 2),
				testSample(t, testSingleMetric, []int{101, 101}, 3),
			),
			limits: limit,
		},
		{
			name: "should collapse metrics into metrics already containg aggregated code",
			inputSamples: append(
				topPriorityTestSamplesWithSizeLowerThanTreshold,
				testSample(t, testSingleMetric, []int{lookup.LookupAggrCode, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 100}, 1),
				testSample(t, testSingleMetric, []int{101, 101}, 3),
			),
			expectedSamples: append(
				topPriorityTestSamplesWithSizeLowerThanTreshold,
				testSample(t, testTwoSingleMetricAggregated, []int{lookup.LookupAggrCode, 100}, 2),
				testSample(t, testSingleMetric, []int{101, 101}, 3),
			),
			limits: limit,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := NewPriorityTable(testSchema, test.limits)
			for _, s := range test.inputSamples {
				assert.NoError(t, table.Add(s))
			}
			assert.NoError(t, table.CollapseSamples())
			samples := []*sample.Sample{}
			for _, s := range table.Samples() {
				samples = append(samples, s)
			}
			assert.ElementsMatch(t, test.expectedSamples, samples)
		})
	}

}

func testSample(t *testing.T, metrics []sample.Metric, dimensionsCodes []int, hitcount int) *sample.Sample {
	s := sample.NewSample(len(dimensionsCodes)*8, len(metrics))
	s.AddHitCount(hitcount - 1)
	for _, c := range dimensionsCodes {
		err := s.Key().AddKeyPart(c, 8)
		assert.NoError(t, err)
	}

	for i, m := range metrics {
		s.Metrics()[i] = m
	}

	for i := range s.Metrics() {
		s.Metrics()[i].Count = float64(hitcount)
	}

	return &s
}

func generateUniqueTestSamples(t *testing.T, numberOfSamples int, dimensionCodesStart []int, hitcount int, metrics []sample.Metric) []*sample.Sample {
	ret := make([]*sample.Sample, 0, numberOfSamples)

	for i := 0; i < numberOfSamples; i++ {
		ret = append(ret, testSample(t, metrics, appendInt(dimensionCodesStart, i), hitcount))
	}
	return ret
}

func appendInt(array []int, number int) []int {
	ret := make([]int, 0, len(array))
	for _, i := range array {
		ret = append(ret, i+number)
	}
	return ret
}

func testLimits(t *testing.T, max int, threshold int) limits.Limits {
	l, err := limits.NewLimits(max, threshold)
	require.NoError(t, err)
	return l
}
