package metrics

import "github.com/nginx/agent/v3/internal/bus"

type (
	// Represents a single entry for a data source, is one of [counter, gauge, histogram, summary].
	DataEntry struct {
		Name        string
		Type        string
		SourceType  string
		Description string
		Values      []DataPoint
	}

	// A single data point for an entry. An entry can have multiple points.
	DataPoint struct {
		Name   string
		Labels map[string]string
		Value  float64
	}
)

func (de *DataEntry) ToBusMessage() *bus.Message {
	return &bus.Message{
		Topic: bus.METRICS_TOPIC,
		Data:  de,
	}
}
