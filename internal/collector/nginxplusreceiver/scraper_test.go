// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"path/filepath"
	"testing"

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

	scraper := newNginxPlusScraper(receivertest.NewNopSettings(component.Type{}), cfg)
	err := scraper.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	_, err = scraper.Scrape(context.Background())
	require.NoError(t, err)

	// To test the nginx.http.response.count metric calculation we need to set the previousLocationZoneResponses &
	// previousSeverZoneResponses then call scrape a second time as the first time it is called the previous responses
	// are set using the API

	/* Initial metrics for location zone:
	{
	  "location_test": {
	    "requests": 34,
	    "responses": {
	      "1xx": 7,
	      "2xx": 31,
	      "3xx": 0,
	      "4xx": 3,
	      "5xx": 0,
	      "codes": {
	        "200": 31,
	        "404": 3
	      },
	      "total": 34
	    },
	    "discarded": 0,
	    "received": 3609,
	    "sent": 23265
	  }
	}
	*/
	scraper.previousLocationZoneResponses = map[string]ResponseStatuses{
		"location_test": {
			oneHundredStatusRange:   3,  // delta 4
			twoHundredStatusRange:   29, // delta 2
			threeHundredStatusRange: 0,
			fourHundredStatusRange:  1, // delta 2
			fiveHundredStatusRange:  0,
		},
	}

	/* Initial metrics for server zone:
	{
	  "test": {
	    "processing": 1,
	    "requests": 32,
	    "responses": {
	      "1xx": 5,
	      "2xx": 29,
	      "3xx": 0,
	      "4xx": 2,
	      "5xx": 0,
	      "codes": {
	        "200": 29,
	        "404": 2
	      },
	      "total": 31
	    },
	    "discarded": 0,
	    "received": 3312,
	    "sent": 21860
	  }
	}
	*/
	scraper.previousServerZoneResponses = map[string]ResponseStatuses{
		"test": {
			oneHundredStatusRange:   3, // delta 2
			twoHundredStatusRange:   0, // delta 29
			threeHundredStatusRange: 0,
			fourHundredStatusRange:  1, // delta 1
			fiveHundredStatusRange:  0,
		},
	}

	scraper.previousHTTPRequestsTotal = 3

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
