package prometheusmetrics

import (
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"

	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type Exporter struct {
	latestMetricReport *proto.MetricsReport
}

func NewExporter(report *proto.MetricsReport) *Exporter {
	return &Exporter{latestMetricReport: report}
}

func (e *Exporter) SetLatestMetricReport(latest *proto.MetricsReport) {
	e.latestMetricReport = latest
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})
	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()
	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}
func metricName(in string) string {
	return strings.Replace(in, ".", "_", -1)
}
func metricLabels(Dimensions []*proto.Dimension) map[string]string {
	m := make(map[string]string)
	for _, dimension := range Dimensions {
		name := metricName(dimension.Name)
		m[name] = dimension.Value
	}
	return m
}

func getValueType(metricName string) prometheus.ValueType {
	calMap := metrics.CalculationMap()

	if value, ok := calMap[metricName]; ok {
		if value == "sum" {
			return prometheus.CounterValue
		} else {
			return prometheus.GaugeValue
		}

	}

	return prometheus.GaugeValue
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, statsEntity := range e.latestMetricReport.Data {
		for _, metric := range statsEntity.Simplemetrics {
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					metricName(metric.Name),
					"Metric Report",
					nil,
					metricLabels(statsEntity.Dimensions),
				),
				getValueType(metric.Name), metric.Value,
			)
		}
	}
}
