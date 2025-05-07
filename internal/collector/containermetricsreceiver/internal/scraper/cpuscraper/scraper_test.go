// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cpuscraper

import (
	"context"
	"path"
	"runtime"
	"testing"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper/internal/cgroup"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/scraper/scrapertest"
)

func TestScrape(t *testing.T) {
	ctx := context.Background()

	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)
	basePath = localDirectory + "/../testdata/good_data/v1/"
	cgroup.CPUStatsPath = localDirectory + "/../testdata/proc/stat"

	scraper := NewScraper(
		ctx,
		scrapertest.NewNopSettings(component.Type{}),
		NewConfig(&config.Config{}),
	)

	err := scraper.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	metrics, err := scraper.Scrape(ctx)
	require.NotNil(t, metrics)
	require.NoError(t, err)
}
