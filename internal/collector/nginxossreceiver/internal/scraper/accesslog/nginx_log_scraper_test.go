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

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	testDataDir = "testdata"
	baseformat  = `$remote_addr - $remote_user [$time_local] "$request"` +
		` $status $body_bytes_sent "$http_referer" "$http_user_agent"` +
		` "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time"` +
		` "$gzip_ratio" "$server_protocol" "$upstream_connect_time""$upstream_header_time"` +
		` "$upstream_response_length" "$upstream_response_time"`
)

func TestAccessLogScraper(t *testing.T) {
	tempDir := t.TempDir()
	var (
		testAccessLogPath = filepath.Join(tempDir, "test.log")
		testDataFilePath  = filepath.Join(testDataDir, "test-access.log")
	)

	cfg, ok := config.CreateDefaultConfig().(*config.Config)
	assert.True(t, ok)
	cfg.InputConfig.Include = []string{testAccessLogPath}
	cfg.AccessLogFormat = baseformat

	accessLogScraper, err := NewScraper(receivertest.NewNopSettings(), cfg)
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
		_, err := NewScraper(receivertest.NewNopSettings(), &config.Config{
			InputConfig: file.Config{},
		})
		require.Error(tt, err)
		assert.Contains(tt, err.Error(), "init stanza pipeline")
	})

	t.Run("log_format error", func(tt *testing.T) {
		dc, ok := config.CreateDefaultConfig().(*config.Config)
		assert.True(t, ok)
		dc.InputConfig.Include = []string{testDataDir}
		_, err := NewScraper(receivertest.NewNopSettings(), dc)
		require.Error(tt, err)
		assert.Contains(tt, err.Error(), "NGINX log format missing")
	})
}

// Copies the contents of one file to another with the given delay. Used to simulate writing log entries to a log file.
// Reason for nolint: we must use testify's assert instead of require,
// for more info see https://github.com/stretchr/testify/issues/772#issuecomment-945166599
// nolint: testifylint
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
