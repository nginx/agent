// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"path/filepath"
	"testing"

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
	require.NoError(t, component.ValidateConfig(cfg))

	scraper, err := newNginxPlusScraper(receivertest.NewNopSettings(), cfg)
	require.NoError(t, err)

	_, err = scraper.scrape(context.Background())
	require.NoError(t, err)

	// To test the nginx.http.response.count metric calculation we need to set the previousLocationZoneResponses &
	// previousSeverZoneResponses then call scrape a second time as the first time it is called the previous responses
	// are set using the API
	scraper.previousLocationZoneResponses = map[string]ResponseStatuses{
		"location_test": {
			oneHundredStatusRange:   3,  // 4
			twoHundredStatusRange:   29, // 2
			threeHundredStatusRange: 0,
			fourHundredStatusRange:  1, // 2
			fiveHundredStatusRange:  0,
		},
	}

	scraper.previousServerZoneResponses = map[string]ResponseStatuses{
		"test": {
			oneHundredStatusRange:   3, // 2
			twoHundredStatusRange:   0, // 29
			threeHundredStatusRange: 0,
			fourHundredStatusRange:  1, // 1
			fiveHundredStatusRange:  0,
		},
	}

	actualMetrics, err := scraper.scrape(context.Background())
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
