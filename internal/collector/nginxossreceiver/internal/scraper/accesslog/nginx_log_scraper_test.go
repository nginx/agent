// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package accesslog

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const testDataDir = "testdata"

func TestAccessLogScraper(t *testing.T) {
	tempDir := t.TempDir()
	var (
		testAccessLogPath = filepath.Join(tempDir, "test.log")
		nginxConfPath     = filepath.Join("..", "..", "..", testDataDir, "integration", "default.conf")
		testDataFilePath  = filepath.Join(testDataDir, "test-access.log")
	)

	cfg := config.CreateDefaultConfig().(*config.Config)
	cfg.InputConfig.Include = []string{testAccessLogPath}
	cfg.NginxConfigPath = nginxConfPath

	accessLogScraper, err := NewScraper(receivertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)

	err = accessLogScraper.Start(context.Background(), componenttest.NewNopHost())
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
		pmetrictest.IgnoreMetricsOrder()))
}

func TestAccessLogScraperError(t *testing.T) {
	t.Run("include config missing", func(tt *testing.T) {
		_, err := NewScraper(receivertest.NewNopCreateSettings(), &config.Config{
			InputConfig: file.Config{},
		})
		require.Error(tt, err)
		assert.Contains(tt, err.Error(), "init stanza pipeline")
	})

	t.Run("log_format error", func(tt *testing.T) {
		dc := config.CreateDefaultConfig().(*config.Config)
		dc.InputConfig.Include = []string{testDataDir}
		_, err := NewScraper(receivertest.NewNopCreateSettings(), dc)
		require.Error(tt, err)
		assert.Contains(tt, err.Error(), "NGINX log format missing")
	})
}

// Copies the contents of one file to another with the given delay. Used to simulate writing log entries to a log file.
func simulateLogging(t *testing.T, sourcePath, destinationPath string, delay time.Duration) {
	t.Helper()

	src, err := os.Open(sourcePath)
	require.NoError(t, err)
	defer src.Close()

	var dest *os.File
	if _, err := os.Stat(destinationPath); os.IsNotExist(err) {
		dest, err = os.Create(destinationPath)
		require.NoError(t, err)
	} else {
		dest, err = os.OpenFile(destinationPath, os.O_RDWR|os.O_APPEND, 0o660)
		require.NoError(t, err)
	}
	defer dest.Close()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		<-time.After(delay)

		logLine := scanner.Text()
		_, err := dest.WriteString(logLine + "\n")
		require.NoError(t, err)
	}
}
