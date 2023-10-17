package prometheus_metrics

import (
	"testing"

	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestExporter(t *testing.T) {
	metricReport1 := &proto.MetricsReport{Meta: &proto.Metadata{MessageId: "123"}}
	metricReport2 := &proto.MetricsReport{Meta: &proto.Metadata{MessageId: "456"}}
	metricReport3 := &proto.MetricsReport{Type: proto.MetricsReport_CACHE_ZONE, Meta: &proto.Metadata{MessageId: "789"}}

	exporter := NewExporter(metricReport1)

	assert.Equal(t, metricReport1, exporter.GetLatestMetricReports()[0])

	exporter.SetLatestMetricReport(&metrics.MetricsReportBundle{Data: []*proto.MetricsReport{metricReport2}})

	assert.Equal(t, metricReport2, exporter.GetLatestMetricReports()[0])

	exporter.SetLatestMetricReport(&metrics.MetricsReportBundle{Data: []*proto.MetricsReport{metricReport2, metricReport3}})

	assert.Equal(t, metricReport2, exporter.GetLatestMetricReports()[0])
	assert.Equal(t, metricReport3, exporter.GetLatestMetricReports()[1])
}

func TestExporter_convertMetricNameToPrometheusFormat(t *testing.T) {
	expected := "test_metric_name"
	actual := convertMetricNameToPrometheusFormat("test.metric.name")

	assert.Equal(t, expected, actual)
}

func TestExporter_convertDimensionsToLabels(t *testing.T) {
	expected := make(map[string]string)
	expected["dimension1"] = "123"
	expected["dimension_2"] = "456"
	expected["dimension_3"] = "789"

	actual := convertDimensionsToLabels([]*proto.Dimension{
		{Name: "dimension1", Value: "123"},
		{Name: "dimension_2", Value: "456"},
		{Name: "dimension.3", Value: "789"},
	})

	assert.Equal(t, expected, actual)

	// Verify empty dimensions

	expected = make(map[string]string)
	actual = convertDimensionsToLabels([]*proto.Dimension{})

	assert.Equal(t, expected, actual)
}

func TestExporter_getValueType(t *testing.T) {
	// Verify avg metric
	expected := prometheus.GaugeValue
	actual := getValueType("system.cpu.idle")

	assert.Equal(t, expected, actual)

	// Verify sum metric
	expected = prometheus.CounterValue
	actual = getValueType("system.io.iops_r")

	assert.Equal(t, expected, actual)

	// Verify boolean metric
	expected = prometheus.GaugeValue
	actual = getValueType("nginx.status")

	assert.Equal(t, expected, actual)
}

func TestExporter_createPrometheusMetric(t *testing.T) {
	metric := &proto.SimpleMetric{
		Name:  "metric.name",
		Value: 123,
	}

	dimensions := []*proto.Dimension{
		{Name: "dimension1", Value: "123"},
		{Name: "dimension_2", Value: "456"},
		{Name: "dimension.3", Value: "789"},
	}

	expected := "Desc{fqName: \"metric_name\", help: \"\", constLabels: {dimension1=\"123\",dimension_2=\"456\",dimension_3=\"789\"}, variableLabels: {}}"
	actual := createPrometheusMetric(metric, dimensions)

	assert.Equal(t, expected, actual.Desc().String())
}
