// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package utils

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/go-resty/resty/v2"
	"github.com/testcontainers/testcontainers-go"

	"github.com/nginx/agent/v3/test/helpers"
)

var MockCollectorStack *helpers.MockCollectorContainers

const envContainer = "Container"

func SetupMetricsTest(tb testing.TB) func(testing.TB) {
	tb.Helper()
	ctx := context.Background()

	if os.Getenv("TEST_ENV") == envContainer {
		setupStackEnvironment(ctx, tb)
	}

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == envContainer {
			helpers.LogAndTerminateStack(
				ctx,
				tb,
				MockCollectorStack,
			)
		}
	}
}

func setupStackEnvironment(ctx context.Context, tb testing.TB) {
	tb.Helper()
	tb.Log("Running tests in a container environment")

	containerNetwork := createContainerNetwork(ctx, tb)
	setupMockCollectorStack(ctx, tb, containerNetwork)
}

func setupMockCollectorStack(ctx context.Context, tb testing.TB, containerNetwork *testcontainers.DockerNetwork) {
	tb.Helper()

	tb.Log("Starting mock collector stack")

	agentConfig := "../../mock/collector/nginx-agent.conf"
	MockCollectorStack = helpers.StartMockCollectorStack(ctx, tb, containerNetwork, agentConfig)
}

func ScrapeCollectorMetricFamilies(t *testing.T, ctx context.Context,
	otelContainer testcontainers.Container,
) map[string]*dto.MetricFamily {
	t.Helper()

	host, _ := otelContainer.Host(ctx)
	port, _ := otelContainer.MappedPort(ctx, "9775")

	address := net.JoinHostPort(host, port.Port())
	url := fmt.Sprintf("http://%s/metrics", address)

	client := resty.New()
	resp, err := client.R().EnableTrace().Get(url)
	if err != nil {
		t.Fatalf("failed to get response from Otel Collector: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode())
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(resp.Body()))
	if err != nil {
		t.Fatalf("failed to parse metrics: %v", err)
	}

	return metricFamilies
}

func SumMetricFamily(metricFamily *dto.MetricFamily) float64 {
	var total float64
	for _, metric := range metricFamily.GetMetric() {
		if value := metricValue(metricFamily, metric); value != nil {
			total += *value
		}
	}

	return total
}

func SumMetricFamilyLabel(metricFamily *dto.MetricFamily, key, val string) float64 {
	var total float64
	for _, metric := range metricFamily.GetMetric() {
		labels := make(map[string]string)
		for _, labelPair := range metric.GetLabel() {
			labels[labelPair.GetName()] = labelPair.GetValue()
		}
		if labels[key] != val {
			continue
		}
		if value := metricValue(metricFamily, metric); value != nil {
			total += *value
		}
	}

	return total
}

func metricValue(metricFamily *dto.MetricFamily, metric *dto.Metric) *float64 {
	switch metricFamily.GetType() {
	case dto.MetricType_COUNTER:
		return getCounterValue(metric)
	case dto.MetricType_GAUGE:
		return getGaugeValue(metric)
	case dto.MetricType_SUMMARY:
		return getSummaryValue(metric)
	case dto.MetricType_UNTYPED:
		return getUntypedValue(metric)
	case dto.MetricType_HISTOGRAM, dto.MetricType_GAUGE_HISTOGRAM:
		return getHistogramValue(metric)
	}

	return nil
}

func getCounterValue(metric *dto.Metric) *float64 {
	if counter := metric.GetCounter(); counter != nil {
		val := counter.GetValue()
		return &val
	}

	return nil
}

func getGaugeValue(metric *dto.Metric) *float64 {
	if gauge := metric.GetGauge(); gauge != nil {
		val := gauge.GetValue()
		return &val
	}

	return nil
}

func getSummaryValue(metric *dto.Metric) *float64 {
	if summary := metric.GetSummary(); summary != nil {
		val := summary.GetSampleSum()
		return &val
	}

	return nil
}

func getUntypedValue(metric *dto.Metric) *float64 {
	if untyped := metric.GetUntyped(); untyped != nil {
		val := untyped.GetValue()
		return &val
	}

	return nil
}

func getHistogramValue(metric *dto.Metric) *float64 {
	if histogram := metric.GetHistogram(); histogram != nil {
		val := histogram.GetSampleSum()
		return &val
	}

	return nil
}
