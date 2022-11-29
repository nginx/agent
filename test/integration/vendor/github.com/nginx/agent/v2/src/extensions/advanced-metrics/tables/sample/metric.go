package sample

// Metric is a simple counter like metric with its summary stats bundled in
type Metric struct {
	Count float64
	Last  float64
	Min   float64
	Max   float64
	Sum   float64
}

func NewMetric(value float64) Metric {
	return Metric{
		Count: 1,
		Last:  value,
		Min:   value,
		Max:   value,
		Sum:   value,
	}
}

func (m *Metric) Add(d float64) {
	m.Last = d
	m.Count++
	m.Sum += d
	if m.Max < d {
		m.Max = d
	}
	if m.Min > d {
		m.Min = d
	}
}

func (m *Metric) AddMetric(m2 Metric) {
	m.Last = m2.Last
	m.Count += m2.Count
	m.Sum += m2.Sum
	if m.Max < m2.Max {
		m.Max = m2.Max
	}
	if m.Min > m2.Sum {
		m.Min = m2.Sum
	}
}
