// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func newTestScraper(t *testing.T, cfg *Config) *CertificateScraper {
	t.Helper()
	settings := receivertest.NewNopSettings(component.MustNewType("certificate"))
	s := newCertificateScraper(settings, cfg)
	require.NoError(t, s.Start(context.Background(), nil))

	return s
}

func TestScrape_FutureExpiry(t *testing.T) {
	notAfter := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertMeta: map[string]config.CertMeta{
			"/etc/nginx/cert.pem": {
				SerialNumber:       "12345",
				CommonName:         "example.com",
				PublicKeyAlgorithm: "ECDSA",
				NotAfter:           notAfter.Unix(),
			},
		},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	rm := metrics.ResourceMetrics().At(0)

	instanceID, ok := rm.Resource().Attributes().Get("instance.id")
	require.True(t, ok)
	require.Equal(t, "test-instance", instanceID.AsString())

	ms := rm.ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, ms.Len())
	require.Equal(t, "nginx.ssl.certificate.expiry.time", ms.At(0).Name())

	dp := ms.At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter.Unix(), dp.IntValue())

	filePathAttr, ok := dp.Attributes().Get("file_path")
	require.True(t, ok)
	require.Equal(t, "/etc/nginx/cert.pem", filePathAttr.AsString())

	commonName, ok := dp.Attributes().Get("subject.common_name")
	require.True(t, ok)
	require.Equal(t, "example.com", commonName.AsString())

	pubKeyAlgo, ok := dp.Attributes().Get("public_key_algorithm")
	require.True(t, ok)
	require.Equal(t, "ECDSA", pubKeyAlgo.AsString())

	serialAttr, ok := dp.Attributes().Get("serial_number")
	require.True(t, ok)
	require.Equal(t, "12345", serialAttr.AsString())
}

func TestScrape_ExpiredCert(t *testing.T) {
	notAfter := time.Now().Add(-7 * 24 * time.Hour).Truncate(time.Second)

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertMeta: map[string]config.CertMeta{
			"/etc/nginx/expired.pem": {
				SerialNumber:       "99999",
				CommonName:         "expired.example.com",
				PublicKeyAlgorithm: "RSA",
				NotAfter:           notAfter.Unix(),
			},
		},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	dp := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter.Unix(), dp.IntValue())
}

func TestScrape_NoCerts(t *testing.T) {
	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertMeta:             nil,
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, metrics.ResourceMetrics().Len(), "Should return empty metrics when no certs configured")
}

func TestScrape_MultipleCerts(t *testing.T) {
	expiredNotAfter := time.Now().Add(-7 * 24 * time.Hour).Truncate(time.Second)

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertMeta: map[string]config.CertMeta{
			"/a.pem": {
				SerialNumber:       "1",
				CommonName:         "one.example.com",
				PublicKeyAlgorithm: "ECDSA",
				NotAfter:           time.Now().Add(10 * 24 * time.Hour).Unix(),
			},
			"/b.pem": {
				SerialNumber:       "2",
				CommonName:         "two.example.com",
				PublicKeyAlgorithm: "RSA",
				NotAfter:           time.Now().Add(60 * 24 * time.Hour).Unix(),
			},
			"/expired.pem": {
				SerialNumber:       "3",
				CommonName:         "expired.example.com",
				PublicKeyAlgorithm: "RSA",
				NotAfter:           expiredNotAfter.Unix(),
			},
		},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	ms := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, ms.Len())
	require.Equal(t, 3, ms.At(0).Gauge().DataPoints().Len(), "Should emit one data point per certificate")
}

func TestScrape_RenewExpiredCert(t *testing.T) {
	expiredNotAfter := time.Now().Add(-7 * 24 * time.Hour).Truncate(time.Second)

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertMeta: map[string]config.CertMeta{
			"/cert.pem": {
				SerialNumber:       "old-serial",
				CommonName:         "example.com",
				PublicKeyAlgorithm: "RSA",
				NotAfter:           expiredNotAfter.Unix(),
			},
		},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	dp := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, expiredNotAfter.Unix(), dp.IntValue())

	// Simulate cert renewal by updating the config with new metadata.
	renewedNotAfter := time.Now().Add(365 * 24 * time.Hour).Truncate(time.Second)
	cfg.CertMeta["/cert.pem"] = config.CertMeta{
		SerialNumber:       "new-serial",
		CommonName:         "example.com",
		PublicKeyAlgorithm: "RSA",
		NotAfter:           renewedNotAfter.Unix(),
	}

	metrics, err = scraper.Scrape(context.Background())
	require.NoError(t, err)
	dp = metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, renewedNotAfter.Unix(), dp.IntValue())

	serialAttr, ok := dp.Attributes().Get("serial_number")
	require.True(t, ok)
	require.Equal(t, "new-serial", serialAttr.AsString())
}
