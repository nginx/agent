// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package stubstatus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
)

const testDataDir = "testdata"

func TestStubStatusScraper(t *testing.T) {
	nginxMock := newMockServer(t)
	defer nginxMock.Close()
	cfg := config.CreateDefaultConfig().(*config.Config)
	cfg.Endpoint = nginxMock.URL + "/status"
	require.NoError(t, component.ValidateConfig(cfg))

	stubStatusScraper := NewScraper(receivertest.NewNopCreateSettings(), cfg)

	err := stubStatusScraper.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	actualMetrics, err := stubStatusScraper.Scrape(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join(testDataDir, "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreTimestamp(),
		pmetrictest.IgnoreMetricsOrder()))
}

func TestStubStatusScraperError(t *testing.T) {
	nginxMock := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/status" {
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte(`Bad status page`))
			return
		}
		rw.WriteHeader(404)
	}))
	t.Run("404", func(t *testing.T) {
		sc := NewScraper(receivertest.NewNopCreateSettings(), &config.Config{
			ClientConfig: confighttp.ClientConfig{
				Endpoint: nginxMock.URL + "/badpath",
			},
		})
		err := sc.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)
		_, err = sc.Scrape(context.Background())
		require.Equal(t, errors.New("expected 200 response, got 404"), err)
	})

	t.Run("parse error", func(t *testing.T) {
		sc := NewScraper(receivertest.NewNopCreateSettings(), &config.Config{
			ClientConfig: confighttp.ClientConfig{
				Endpoint: nginxMock.URL + "/status",
			},
		})
		err := sc.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)
		_, err = sc.Scrape(context.Background())
		require.ErrorContains(t, err, "Bad status page")
	})
	nginxMock.Close()
}

func TestScraperFailedStart(t *testing.T) {
	sc := NewScraper(receivertest.NewNopCreateSettings(), &config.Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "localhost:8080",
			TLSSetting: configtls.ClientConfig{
				Config: configtls.Config{
					CAFile: "/non/existent",
				},
			},
		},
	})
	err := sc.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err)
}

func newMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/status" {
			rw.WriteHeader(200)
			_, err := rw.Write([]byte(`Active connections: 291
server accepts handled requests
 16630948 16630946 31070465
Reading: 6 Writing: 179 Waiting: 106
`))
			require.NoError(t, err)
			return
		}
		rw.WriteHeader(404)
	}))
}
