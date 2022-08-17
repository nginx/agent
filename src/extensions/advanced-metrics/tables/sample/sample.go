package sample

import (
	"errors"
	"fmt"
)

type Sample struct {
	key      SampleKey
	metrics  []Metric
	hitCount int
}

func NewSample(keySize int, metricsSize int) Sample {
	sample := Sample{
		key:     NewSampleKey(keySize),
		metrics: make([]Metric, metricsSize),

		hitCount: 1,
	}
	return sample
}

func (s *Sample) AddHitCount(hitCount int) {
	s.hitCount += hitCount
}

func (s *Sample) HitCount() int {
	return s.hitCount
}

func (s *Sample) Key() *SampleKey {
	return &s.key
}

func (s *Sample) Metrics() []Metric {
	return s.metrics
}

func (s *Sample) Metric(i int) (Metric, error) {
	if i >= len(s.metrics) {
		return Metric{}, errors.New("metric index out of range")
	}
	return s.metrics[i], nil
}

func (s *Sample) AddSample(other *Sample) error {
	if len(s.metrics) != len(other.metrics) {
		return errors.New("sample metrics number mismatch")
	}

	if s.key.AsStringKey() != other.key.AsStringKey() {
		return errors.New("samples key differs")
	}

	for i := range other.metrics {
		s.metrics[i].AddMetric(other.metrics[i])
	}

	s.hitCount += other.hitCount

	return nil
}

func (s *Sample) SetMetric(metricIndex int, value float64) error {
	if metricIndex >= len(s.metrics) {
		return fmt.Errorf("metric index: %d out of range: [0,%d)", metricIndex, len(s.metrics))
	}

	s.metrics[metricIndex] = NewMetric(value)
	return nil
}
