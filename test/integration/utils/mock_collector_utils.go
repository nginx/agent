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
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"

	"github.com/go-resty/resty/v2"
	"github.com/testcontainers/testcontainers-go"

	"github.com/nginx/agent/v3/test/helpers"
)

var MockCollectorStack *helpers.MockCollectorContainers

const (
	envContainer  = "Container"
	tickerTime    = 1 * time.Second
	timeoutTime   = 100 * time.Second
	plusImagePath = "/nginx-plus/agent"
)

type LabelFilter struct {
	Key    string
	Values []string
}

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

	containerNetwork := CreateContainerNetwork(ctx, tb)
	setupMockCollectorStack(ctx, tb, containerNetwork)
}

func setupMockCollectorStack(ctx context.Context, tb testing.TB, containerNetwork *testcontainers.DockerNetwork) {
	tb.Helper()

	tb.Log("Starting mock collector stack")

	nginxConfPath := "../../config/nginx/nginx-for-metric-testing.conf"
	if os.Getenv("IMAGE_PATH") == plusImagePath {
		nginxConfPath = "../../config/nginx/nginx-plus-for-metric-testing.conf"
	}
	agentConfig := "../../mock/collector/nginx-agent.conf"

	params := &helpers.Parameters{
		NginxConfigPath:      nginxConfPath,
		NginxAgentConfigPath: agentConfig,
		LogMessage:           "Starting NGINX Agent",
	}

	MockCollectorStack = helpers.StartMockCollectorStack(ctx, tb, containerNetwork)
	MockCollectorStack.Agent = helpers.StartContainer(ctx, tb, containerNetwork, params)
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

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(resp.Body()))
	if err != nil {
		t.Fatalf("failed to parse metrics: %v", err)
	}

	return metricFamilies
}

func GenerateMetrics(ctx context.Context, t *testing.T, container testcontainers.Container,
	requestCount int, expectedCode string,
) {
	t.Helper()

	t.Logf("Generating %d requests with expected response code %s", requestCount, expectedCode)

	var url string
	switch expectedCode {
	case "1xx":
		url = "http://127.0.0.1:80/1xx"
	case "2xx":
		url = "http://127.0.0.1:80/2xx"
	case "3xx":
		url = "http://127.0.0.1:80/3xx"
	case "4xx":
		url = "http://127.0.0.1:80/4xx"
	case "5xx":
		url = "http://127.0.0.1:80/5xx"

	default:
		url = "http://127.0.0.1/"
	}

	for range requestCount {
		_, _, err := container.Exec(
			ctx,
			[]string{"curl", "-s", url},
		)
		if err != nil {
			t.Fatalf("failed to curl nginx: %s", err)
		}
	}
}

func PollingForMetrics(t *testing.T, ctx context.Context, metricName string,
	labelFilter LabelFilter, baselineValues []float64,
) []float64 {
	t.Helper()

	pollCtx, cancel := context.WithTimeout(ctx, timeoutTime)
	defer cancel()

	ticker := time.NewTicker(tickerTime)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			t.Fatalf("timed out waiting for metric %s to be greater than %v", metricName, baselineValues)
			return nil
		case <-ticker.C:
			family := ScrapeCollectorMetricFamilies(t, ctx, MockCollectorStack.Otel)[metricName]
			if family == nil {
				t.Logf("Metric %s not found, retrying...", metricName)
				continue
			}

			values, changed := metricValueChangeCheck(t, family, labelFilter.Key, labelFilter.Values, baselineValues)
			if !changed {
				continue
			}

			return values
		}
	}
}

func WaitUntilNextScrapeCycle(t *testing.T, ctx context.Context) {
	t.Helper()

	waitCtx, cancel := context.WithTimeout(ctx, timeoutTime)
	defer cancel()

	ticker := time.NewTicker(tickerTime)
	defer ticker.Stop()

	prevScrapeValue, err := requestCountMetric(t, ctx)
	if err != nil {
		t.Fatalf("Failed to get initial scrape value: %v", err)
		return
	}

	for {
		select {
		case <-waitCtx.Done():
			t.Fatalf("Timed out waiting for new scrape cycle")
			return
		case <-ticker.C:
			currentMetric, requestErr := requestCountMetric(t, ctx)
			if requestErr != nil {
				continue
			}

			if currentMetric != prevScrapeValue {
				t.Log("Successfully detected new scrape cycle")

				return
			}
		}
	}
}

func WaitForMetricsToExist(t *testing.T, ctx context.Context) {
	t.Helper()

	waitCtx, cancel := context.WithTimeout(ctx, timeoutTime)
	defer cancel()

	ticker := time.NewTicker(tickerTime)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			t.Fatal("Timed out waiting for NGINX metrics to exist")
			return
		case <-ticker.C:
			family := ScrapeCollectorMetricFamilies(t, ctx, MockCollectorStack.Otel)["nginx_http_request_count"]
			if family != nil {
				t.Log("NGINX metrics found")
				return
			}
			t.Log("NGINX metrics not found, retrying...")
		}
	}
}

func requestCountMetric(t *testing.T, ctx context.Context) (float64, error) {
	t.Helper()

	family := ScrapeCollectorMetricFamilies(t, ctx, MockCollectorStack.Otel)["nginx_http_request_count"]
	if family == nil {
		return 0, fmt.Errorf("metric nginx_http_request_count not found: %v", family)
	}

	return SumMetricFamily(family), nil
}

// metricValueChangeCheck checks if the metric values in a MetricFamily have changed
func metricValueChangeCheck(t *testing.T, family *dto.MetricFamily, labelKey string,
	labelValues []string, baselineValues []float64) (
	[]float64, bool,
) {
	t.Helper()

	if len(family.GetMetric()) == 1 {
		value, changed := checkSingleMetricValue(t, family, baselineValues[0])
		if changed {
			return []float64{value}, true
		}
	}

	if len(family.GetMetric()) > 1 {
		values, allChanged := checkLabeledMetricValue(t, family, labelKey, labelValues, baselineValues)
		if allChanged {
			return values, true
		}
	}

	return []float64{0}, false
}
