// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package accesslog

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"

	"github.com/nginx/agent/v3/internal/collector/nginxreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxreceiver/internal/model"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	testDataDir = "./testdata"
	baseformat  = `$remote_addr - $remote_user [$time_local] "$request"` +
		` $status $body_bytes_sent "$http_referer" "$http_user_agent"` +
		` "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time"` +
		` "$gzip_ratio" "$server_protocol" "$upstream_connect_time""$upstream_header_time"` +
		` "$upstream_response_length" "$upstream_response_time"`
)

func TestAccessLogScraper_ID(t *testing.T) {
	nls := &NginxLogScraper{}
	assert.Equal(t, "nginx", nls.ID().Type().String())
}

func TestAccessLogScraper(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	var (
		testAccessLogPath = filepath.Join(tempDir, "test.log")
		testDataFilePath  = filepath.Join(testDataDir, "test-access.log")
	)

	cfg, ok := config.CreateDefaultConfig().(*config.Config)
	assert.True(t, ok)
	cfg.AccessLogs = []config.AccessLog{
		{
			LogFormat: baseformat,
			FilePath:  testAccessLogPath,
		},
	}

	accessLogScraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)
	defer func() {
		shutdownError := accessLogScraper.Shutdown(ctx)
		require.NoError(t, shutdownError)
	}()

	err := accessLogScraper.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	go simulateLogging(t, testDataFilePath, testAccessLogPath, 250*time.Millisecond)
	<-time.After(cfg.CollectionInterval)

	actualMetrics, err := accessLogScraper.Scrape(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join(testDataDir, "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreTimestamp(),
		pmetrictest.IgnoreMetricsOrder(),
		pmetrictest.IgnoreResourceAttributeValue("instance.id")))
}

func TestAccessLogScraperRouteRequestCounts(t *testing.T) {
	testCases := []struct {
		expectedPoints map[routeKey]int64
		name           string
		entries        []model.NginxAccessItem
	}{
		{
			name: "Test 1: single route increments count and handles empty route kind",
			entries: []model.NginxAccessItem{
				{
					RouteName:        "checkout",
					RouteNamespace:   "default",
					RouteKind:        "",
					GatewayName:      "edge",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
				{
					RouteName:        "checkout",
					RouteNamespace:   "default",
					RouteKind:        "",
					GatewayName:      "edge",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
			},
			expectedPoints: map[routeKey]int64{
				{
					name:        "checkout",
					namespace:   "default",
					kind:        "",
					gwName:      "edge",
					gwNamespace: "default",
					gwClass:     "nginx",
				}: 2,
			},
		},
		{
			name: "Test 2: different route names are isolated",
			entries: []model.NginxAccessItem{
				{
					RouteName:        "coffee",
					RouteNamespace:   "default",
					RouteKind:        "HTTPRoute",
					GatewayName:      "gateway",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
				{
					RouteName:        "coffee",
					RouteNamespace:   "default",
					RouteKind:        "HTTPRoute",
					GatewayName:      "gateway",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
				{
					RouteName:        "tea",
					RouteNamespace:   "default",
					RouteKind:        "HTTPRoute",
					GatewayName:      "gateway",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
			},
			expectedPoints: map[routeKey]int64{
				{
					name:        "coffee",
					namespace:   "default",
					kind:        "HTTPRoute",
					gwName:      "gateway",
					gwNamespace: "default",
					gwClass:     "nginx",
				}: 2,
				{
					name:        "tea",
					namespace:   "default",
					kind:        "HTTPRoute",
					gwName:      "gateway",
					gwNamespace: "default",
					gwClass:     "nginx",
				}: 1,
			},
		},
		{
			name: "Test 3: same route name in different namespaces is isolated",
			entries: []model.NginxAccessItem{
				{
					RouteName:        "checkout",
					RouteNamespace:   "default",
					RouteKind:        "HTTPRoute",
					GatewayName:      "edge",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
				{
					RouteName:        "checkout",
					RouteNamespace:   "other",
					RouteKind:        "HTTPRoute",
					GatewayName:      "edge",
					GatewayNamespace: "default",
					GatewayClass:     "nginx",
				},
			},
			expectedPoints: map[routeKey]int64{
				{
					name:        "checkout",
					namespace:   "default",
					kind:        "HTTPRoute",
					gwName:      "edge",
					gwNamespace: "default",
					gwClass:     "nginx",
				}: 1,
				{
					name:        "checkout",
					namespace:   "other",
					kind:        "HTTPRoute",
					gwName:      "edge",
					gwNamespace: "default",
					gwClass:     "nginx",
				}: 1,
			},
		},
		{
			name: "Test 4: entries without route information do not emit route data points",
			entries: []model.NginxAccessItem{
				{
					Status: "200",
				},
			},
			expectedPoints: make(map[routeKey]int64),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cfg, ok := config.CreateDefaultConfig().(*config.Config)
			require.True(t, ok)

			scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

			for i := range tc.entries {
				item := tc.entries[i]
				scraper.ConsumerCallback(ctx, []*entry.Entry{{Body: &item}})
			}

			metrics, err := scraper.Scrape(ctx)
			require.NoError(t, err)

			points := routeDataPoints(t, metrics)
			require.Len(t, points, len(tc.expectedPoints))
			for route, expectedCount := range tc.expectedPoints {
				point, found := points[route]
				require.True(t, found, "expected metric data point for route %+v", route)
				assert.Equal(t, expectedCount, point.IntValue())
			}
		})
	}
}

func TestAccessLogScraperRouteRequestCounts_Cumulative(t *testing.T) {
	ctx := context.Background()
	cfg, ok := config.CreateDefaultConfig().(*config.Config)
	require.True(t, ok)

	scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

	item := model.NginxAccessItem{
		RouteName:        "coffee",
		RouteNamespace:   "default",
		RouteKind:        "HTTPRoute",
		GatewayName:      "gateway",
		GatewayNamespace: "default",
		GatewayClass:     "nginx",
	}

	expectedKey := routeKey{
		name:        "coffee",
		namespace:   "default",
		kind:        "HTTPRoute",
		gwName:      "gateway",
		gwNamespace: "default",
		gwClass:     "nginx",
	}

	scraper.ConsumerCallback(ctx, []*entry.Entry{{Body: &item}})
	firstMetrics, err := scraper.Scrape(ctx)
	require.NoError(t, err)
	firstPoints := routeDataPoints(t, firstMetrics)
	require.Contains(t, firstPoints, expectedKey)
	assert.Equal(t, int64(1), firstPoints[expectedKey].IntValue())

	scraper.ConsumerCallback(ctx, []*entry.Entry{{Body: &item}, {Body: &item}, {Body: &item}})
	secondMetrics, err := scraper.Scrape(ctx)
	require.NoError(t, err)
	secondPoints := routeDataPoints(t, secondMetrics)
	require.Contains(t, secondPoints, expectedKey)
	require.Equal(t, int64(4), secondPoints[expectedKey].IntValue())
}

// routeDataPoints returns the nginx.http.requests data points keyed by route.
func routeDataPoints(t *testing.T, metrics pmetric.Metrics) map[routeKey]pmetric.NumberDataPoint {
	t.Helper()

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	rm := metrics.ResourceMetrics().At(0)
	require.Equal(t, 1, rm.ScopeMetrics().Len())

	requestMetric, found := findMetric(rm.ScopeMetrics().At(0).Metrics(), "nginx.http.requests")
	if !found {
		return make(map[routeKey]pmetric.NumberDataPoint)
	}
	require.Equal(t, pmetric.MetricTypeSum, requestMetric.Type())

	points := make(map[routeKey]pmetric.NumberDataPoint, requestMetric.Sum().DataPoints().Len())
	for _, dp := range requestMetric.Sum().DataPoints().All() {
		var key routeKey
		dp.Attributes().Range(func(k string, v pcommon.Value) bool {
			switch k {
			case "nginx.route.name":
				key.name = v.Str()
			case "nginx.route.namespace":
				key.namespace = v.Str()
			case "nginx.route.kind":
				key.kind = v.Str()
			case "nginx.gateway.name":
				key.gwName = v.Str()
			case "nginx.gateway.namespace":
				key.gwNamespace = v.Str()
			case "nginx.gateway.class":
				key.gwClass = v.Str()
			}

			return true
		})

		_, duplicate := points[key]
		require.False(t, duplicate, "unexpected duplicate data point for route %+v", key)
		points[key] = dp
	}

	return points
}

func findMetric(metrics pmetric.MetricSlice, name string) (pmetric.Metric, bool) {
	for _, metric := range metrics.All() {
		if metric.Name() == name {
			return metric, true
		}
	}

	return pmetric.Metric{}, false
}

// Copies the contents of one file to another with the given delay. Used to simulate writing log entries to a log file.
// Reason for nolint: we must use testify's assert instead of require,
// for more info see https://github.com/stretchr/testify/issues/772#issuecomment-945166599
//
//nolint:testifylint // we must use testify's assert instead of require
func simulateLogging(t *testing.T, sourcePath, destinationPath string, writeDelay time.Duration) {
	t.Helper()

	src, err := os.Open(sourcePath)
	assert.NoError(t, err)
	defer src.Close()

	var dest *os.File
	if _, fileCheckErr := os.Stat(destinationPath); os.IsNotExist(fileCheckErr) {
		dest, fileCheckErr = os.Create(destinationPath)
		assert.NoError(t, fileCheckErr)
	} else {
		dest, fileCheckErr = os.OpenFile(destinationPath, os.O_RDWR|os.O_APPEND, 0o660)
		assert.NoError(t, fileCheckErr)
	}
	defer dest.Close()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		<-time.After(writeDelay)

		logLine := scanner.Text()
		_, writeErr := dest.WriteString(logLine + "\n")
		assert.NoError(t, writeErr)
	}
}
