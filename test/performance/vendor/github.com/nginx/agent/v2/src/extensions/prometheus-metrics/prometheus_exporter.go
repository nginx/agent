package prometheus_metrics

import (
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

type Exporter struct {
	latestMetricReports *metrics.MetricsReportBundle
}

func NewExporter(report *proto.MetricsReport) *Exporter {
	return &Exporter{latestMetricReports: &metrics.MetricsReportBundle{Data: []*proto.MetricsReport{report}}}
}

func (e *Exporter) SetLatestMetricReport(latest *metrics.MetricsReportBundle) {
	e.latestMetricReports = latest
}

func (e *Exporter) GetLatestMetricReports() (reports []*proto.MetricsReport) {
	for _, report := range e.latestMetricReports.Data {
		reports = append(reports, report)
	}
	return
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

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, report := range e.latestMetricReports.Data {
		for _, statsEntity := range report.Data {
			for _, metric := range statsEntity.Simplemetrics {
				ch <- createPrometheusMetric(metric, statsEntity.GetDimensions())
			}
		}
	}
}

func createPrometheusMetric(metric *proto.SimpleMetric, Dimensions []*proto.Dimension) prometheus.Metric {
	return prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			convertMetricNameToPrometheusFormat(metric.Name),
			"",
			nil,
			convertDimensionsToLabels(Dimensions),
		),
		getValueType(metric.Name), metric.Value,
	)
}

func convertMetricNameToPrometheusFormat(metricName string) string {
	return strings.Replace(metricName, ".", "_", -1)
}

func convertDimensionsToLabels(Dimensions []*proto.Dimension) map[string]string {
	m := make(map[string]string)
	for _, dimension := range Dimensions {
		name := convertMetricNameToPrometheusFormat(dimension.Name)
		m[name] = dimension.Value
	}
	return m
}

func getValueType(metricName string) prometheus.ValueType {
	calMap := metrics.GetCalculationMap()

	if value, ok := calMap[metricName]; ok {
		if value == "sum" {
			return prometheus.CounterValue
		}
	}

	return prometheus.GaugeValue
}
