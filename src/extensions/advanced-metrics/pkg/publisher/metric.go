package publisher

import "github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"

// Dimension holds dimension name and value.
type Dimension struct {
	Name  string
	Value string
}

// MetricValues holds metric min, max, count, total values.
type MetricValues = sample.Metric

// Metric defines metric name and values.
type Metric struct {
	Name   string
	Values MetricValues
}

// MetricSet defines metrics and dimensions associated for metrics.
type MetricSet struct {
	Dimensions []Dimension
	Metrics    []Metric
}
