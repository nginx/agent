// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"path/filepath"
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
