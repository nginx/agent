package sample

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSamplesAdd(t *testing.T) {
	sampleTable := NewSampleTable()

	key := []byte{0x1}
	metricValue := float64(111)
	testSample := newTestSample(t, key, metricValue)

	err := sampleTable.Add(testSample)
	assert.NoError(t, err)
	assert.Equal(t, 1, sampleTable.Len())

	samples := []Sample{}
	sampleTable.Range(func(s *Sample) {
		samples = append(samples, *s)
	})

	expectedSamples := []Sample{newTestSampleWithMetrics(t, key, 1,
		Metric{
			Count: 1,
			Last:  metricValue,
			Min:   metricValue,
			Max:   metricValue,
			Sum:   metricValue,
		},
	)}

	assert.ElementsMatch(t, expectedSamples, samples)

}

func TestSamplesAddMultipleSamples(t *testing.T) {
	sampleTable := NewSampleTable()

	key := []byte{0x1}
	metricValue := float64(111)
	testSample := newTestSample(t, key, metricValue)

	err := sampleTable.Add(testSample)
	assert.NoError(t, err)
	assert.Equal(t, 1, sampleTable.Len())

	key2 := []byte{0x2}
	metricValue2 := float64(111)
	testSample2 := newTestSample(t, key2, metricValue2)

	err = sampleTable.Add(testSample2)
	assert.NoError(t, err)
	assert.Equal(t, 2, sampleTable.Len())

	samples := []Sample{}
	sampleTable.Range(func(s *Sample) {
		samples = append(samples, *s)
	})

	expectedSamples := []Sample{
		newTestSampleWithMetrics(t, key, 1, Metric{
			Count: 1,
			Last:  metricValue,
			Min:   metricValue,
			Max:   metricValue,
			Sum:   metricValue,
		}),
		newTestSampleWithMetrics(t, key2, 1, Metric{
			Count: 1,
			Last:  metricValue2,
			Min:   metricValue2,
			Max:   metricValue2,
			Sum:   metricValue2,
		}),
	}

	assert.ElementsMatch(t, expectedSamples, samples)
}

func TestSamplesAddShouldUpdateExistingMetric(t *testing.T) {
	sampleTable := NewSampleTable()

	key := []byte{0x1}
	metricValue := float64(111)
	testSample := newTestSample(t, key, metricValue)

	err := sampleTable.Add(testSample)
	assert.NoError(t, err)
	assert.Equal(t, 1, sampleTable.Len())

	key = []byte{0x1}
	metricValue = float64(111)
	testSample = newTestSample(t, key, metricValue)

	err = sampleTable.Add(testSample)
	assert.NoError(t, err)
	assert.Equal(t, 1, sampleTable.Len())

	key2 := []byte{0x2}
	metricValue2 := float64(114)
	testSample = newTestSample(t, key2, metricValue2)

	err = sampleTable.Add(testSample)
	assert.NoError(t, err)
	assert.Equal(t, 2, sampleTable.Len())

	samples := []Sample{}
	sampleTable.Range(func(s *Sample) {
		samples = append(samples, *s)
	})

	expectedSamples := []Sample{
		newTestSampleWithMetrics(t, key, 2, Metric{
			Count: 2,
			Last:  metricValue,
			Min:   metricValue,
			Max:   metricValue,
			Sum:   metricValue * 2,
		}),
		newTestSampleWithMetrics(t, key2, 1, Metric{
			Count: 1,
			Last:  metricValue2,
			Min:   metricValue2,
			Max:   metricValue2,
			Sum:   metricValue2,
		}),
	}

	assert.ElementsMatch(t, expectedSamples, samples)
}

func TestSamplesAddThreadSafe(t *testing.T) {

	const repetitions = 100
	key := []byte{0x1}
	metricValue := float64(111)
	key2 := []byte{0x2}
	metricValue2 := float64(114)

	sampleTable := NewSampleTable()

	wg := sync.WaitGroup{}
	wg.Add(repetitions)

	for i := 0; i < repetitions; i++ {
		go func() {
			defer wg.Done()
			key := []byte{0x1}
			metricValue := float64(111)
			testSample := newTestSample(t, key, metricValue)
			err := sampleTable.Add(testSample)
			assert.NoError(t, err)

			key2 := []byte{0x2}
			metricValue2 := float64(114)
			testSample = newTestSample(t, key2, metricValue2)
			err = sampleTable.Add(testSample)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	samples := []Sample{}
	sampleTable.Range(func(s *Sample) {
		samples = append(samples, *s)
	})

	expectedSamples := []Sample{
		newTestSampleWithMetrics(t, key, repetitions, Metric{
			Count: repetitions,
			Last:  metricValue,
			Min:   metricValue,
			Max:   metricValue,
			Sum:   metricValue * repetitions,
		}),
		newTestSampleWithMetrics(t, key2, repetitions, Metric{
			Count: repetitions,
			Last:  metricValue2,
			Min:   metricValue2,
			Max:   metricValue2,
			Sum:   metricValue2 * repetitions,
		}),
	}

	assert.ElementsMatch(t, expectedSamples, samples)
}

func newTestSample(t *testing.T, key []byte, metrics ...float64) Sample {
	sample := NewSample(len(key)*8, len(metrics))
	for i, v := range metrics {
		err := sample.SetMetric(i, v)
		assert.NoError(t, err)
	}

	for _, k := range key {
		err := sample.Key().AddKeyPart(int(k), 8)
		assert.NoError(t, err)
	}

	return sample
}

func newTestSampleWithMetrics(t *testing.T, key []byte, hitcount int, metrics ...Metric) Sample {
	sample := NewSample(len(key)*8, len(metrics))
	sample.metrics = metrics
	sample.hitCount = hitcount
	for _, k := range key {
		err := sample.Key().AddKeyPart(int(k), 8)
		assert.NoError(t, err)
	}

	return sample
}
