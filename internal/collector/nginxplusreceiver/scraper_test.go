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
	cfg.Endpoint = nginxPlusMock.URL + "/api"
	require.NoError(t, component.ValidateConfig(cfg))

	scraper, err := newNginxPlusScraper(receivertest.NewNopSettings(), cfg)
	require.NoError(t, err)

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
	),
	)
}
