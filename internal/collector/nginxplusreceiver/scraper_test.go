// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/record"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestScraper(t *testing.T) {
	nginxPlusMock := helpers.NewMockNGINXPlusAPIServer(t)
	defer nginxPlusMock.Close()

	cfg, ok := createDefaultConfig().(*Config)
	assert.True(t, ok)
	cfg.APIDetails.URL = nginxPlusMock.URL + "/api"

	tmpDir := t.TempDir()
	_, cert := helpers.GenerateSelfSignedCert(t)

	caContents := helpers.Cert{Name: "ca.pem", Type: "CERTIFICATE", Contents: cert}
	caFile := helpers.WriteCertFiles(t, tmpDir, caContents)
	t.Logf("Ca File: %s", caFile)

	cfg.APIDetails.Ca = caFile

	scraper := newNginxPlusScraper(receivertest.NewNopSettings(component.Type{}), cfg)
	err := scraper.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	_, err = scraper.Scrape(context.Background())
	require.NoError(t, err)

	// To test the nginx.http.response.count metric calculation we need to set the previousLocationZoneResponses &
	// previousSeverZoneResponses then call scrape a second time as the first time it is called the previous responses
	// are set using the API
	scraper.locationZoneMetrics.PreviousLocationZoneResponses = map[string]record.ResponseStatuses{
		"location_test": {
			OneHundredStatusRange:   3,  // 4
			TwoHundredStatusRange:   29, // 2
			ThreeHundredStatusRange: 0,
			FourHundredStatusRange:  1, // 2
			FiveHundredStatusRange:  0,
		},
	}

	scraper.serverZoneMetrics.PreviousServerZoneResponses = map[string]record.ResponseStatuses{
		"test": {
			OneHundredStatusRange:   3, // 2
			TwoHundredStatusRange:   0, // 29
			ThreeHundredStatusRange: 0,
			FourHundredStatusRange:  1, // 1
			FiveHundredStatusRange:  0,
		},
	}

	scraper.locationZoneMetrics.PreviousLocationZoneRequests = map[string]int64{
		"location_test": 30, // 5
	}

	scraper.serverZoneMetrics.PreviousServerZoneRequests = map[string]int64{
		"test": 29, // 3
	}

	scraper.httpMetrics.PreviousHTTPRequestsTotal = 3

	actualMetrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "expected.yaml")
	expectedMetrics, err := golden.ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(
		expectedMetrics,
		actualMetrics,
		pmetrictest.IgnoreStartTimestamp(),
		pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreTimestamp(),
		pmetrictest.IgnoreMetricsOrder(),
		pmetrictest.IgnoreResourceAttributeValue("instance.id")),
	)
}

func TestScraper_InitRetryOnFailure(t *testing.T) {
	ctx := context.Background()

	// statsAvailable controls whether the mock server serves real stats.
	// Starts false → first GetStats fails; set to true → subsequent calls succeed.
	var statsAvailable atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/":
			// Always serve — plusapi.WithMaxAPIVersion() calls this during Start().
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[1,2,3,4,5,6,7,8,9]`))

			return
		case "/api/9/":
			// Always serve — client uses this to discover available sub-endpoints.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(
				`["nginx","processes","connections","slabs","http","stream","resolvers","ssl","workers"]`,
			))

			return
		}

		if !statsAvailable.Load() {
			// Simulate Plus API not yet ready.
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Minimal valid responses — enough for GetStats to succeed and for
		// NewHTTPMetrics / NewLocationZoneMetrics / NewServerZoneMetrics to initialize
		switch r.URL.Path {
		case "/api/9/stream", "/api/9/workers":
			// These endpoints return JSON arrays; empty array = no sub-resources.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	cfg, ok := createDefaultConfig().(*Config)
	require.True(t, ok)
	cfg.APIDetails.URL = server.URL + "/api"

	scraper := newNginxPlusScraper(receivertest.NewNopSettings(component.Type{}), cfg)
	require.NoError(t, scraper.Start(ctx, componenttest.NewNopHost()))

	// ── First scrape: Plus API unavailable ──────────────────────────────────
	// Expect an error to be returned (not a panic), and helpers to stay nil
	// so the next tick can retry initialisation.
	_, err := scraper.Scrape(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize plus metrics helpers")
	assert.Nil(t, scraper.httpMetrics, "httpMetrics must stay nil after failed init")
	assert.Nil(t, scraper.locationZoneMetrics, "locationZoneMetrics must stay nil after failed init")
	assert.Nil(t, scraper.serverZoneMetrics, "serverZoneMetrics must stay nil after failed init")

	// ── Plus API becomes available ──────────────────────────────────────────
	statsAvailable.Store(true)

	// ── Second scrape: initialisation succeeds ──────────────────────────────
	// Helpers must be set and no panic must occur
	_, err = scraper.Scrape(ctx)
	require.NoError(t, err)
	assert.NotNil(t, scraper.httpMetrics, "httpMetrics must be set after successful init")
	assert.NotNil(t, scraper.locationZoneMetrics, "locationZoneMetrics must be set after successful init")
	assert.NotNil(t, scraper.serverZoneMetrics, "serverZoneMetrics must be set after successful init")
}
