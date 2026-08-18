// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// writeTempCert writes a self-signed x509 cert with the given NotAfter to a new temp file
// and returns the file path. The file is cleaned up when the test ends.
func writeTempCert(t *testing.T, notAfter time.Time, commonName string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "*.crt")
	require.NoError(t, err)
	defer f.Close()

	appendCertToFile(t, f, notAfter, commonName, 1)

	return f.Name()
}

func appendCertToFile(t *testing.T, f *os.File, notAfter time.Time, commonName string, serial int64) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func newTestScraper(t *testing.T, cfg *Config) *CertificateScraper {
	t.Helper()
	settings := receivertest.NewNopSettings(component.MustNewType("certificate"))
	s := newCertificateScraper(settings, cfg)
	require.NoError(t, s.Start(context.Background(), nil))

	return s
}

func TestScrape_FutureExpiry(t *testing.T) {
	notAfter := time.Now().Add(30 * 24 * time.Hour)
	path := writeTempCert(t, notAfter, "example.com")

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{path},
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
	require.Equal(t, "nginx.ssl.certificate.expiry", ms.At(0).Name())

	dp := ms.At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter.Unix(), dp.IntValue(), "expiry should be the cert's NotAfter Unix timestamp")

	filePathAttr, ok := dp.Attributes().Get("file_path")
	require.True(t, ok)
	require.Equal(t, path, filePathAttr.AsString())

	commonName, ok := dp.Attributes().Get("subject.common_name")
	require.True(t, ok)
	require.Equal(t, "example.com", commonName.AsString())

	pubKeyAlgo, ok := dp.Attributes().Get("public_key_algorithm")
	require.True(t, ok)
	require.Equal(t, "ECDSA", pubKeyAlgo.AsString())
}

func TestScrape_ExpiredCert(t *testing.T) {
	notAfter := time.Now().Add(-7 * 24 * time.Hour)
	path := writeTempCert(t, notAfter, "expired.example.com")

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{path},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	dp := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter.Unix(), dp.IntValue(), "expiry should be the cert's NotAfter Unix timestamp")
}

func TestScrape_NoCerts(t *testing.T) {
	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        nil,
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, metrics.ResourceMetrics().Len(), "Should return empty metrics when no certs configured")
}

func TestScrape_MultipleCerts(t *testing.T) {
	path1 := writeTempCert(t, time.Now().Add(10*24*time.Hour), "one.example.com")
	path2 := writeTempCert(t, time.Now().Add(60*24*time.Hour), "two.example.com")

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{path1, path2},
	}
	scraper := newTestScraper(t, cfg)

	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	ms := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	require.Equal(t, 1, ms.Len())
	require.Equal(t, 2, ms.At(0).Gauge().DataPoints().Len(), "Should emit one data point per certificate")
}

func TestScrape_MissingFile(t *testing.T) {
	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{filepath.Join(t.TempDir(), "nonexistent.crt")},
	}
	scraper := newTestScraper(t, cfg)

	// Missing file is logged and skipped — no error, no metrics
	metrics, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, metrics.ResourceMetrics().Len())
}

func TestScrape_CacheHit(t *testing.T) {
	// Verify that after Start, multiple Scrape calls return correct data from cache.
	notAfter := time.Now().Add(30 * 24 * time.Hour)
	path := writeTempCert(t, notAfter, "cached.example.com")

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{path},
	}
	scraper := newTestScraper(t, cfg)

	for range 2 {
		metrics, err := scraper.Scrape(context.Background())
		require.NoError(t, err)
		dps := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints()
		require.Equal(t, 1, dps.Len())
	}
}

func TestScrape_DoesNotReReadOnRenewal(t *testing.T) {
	// After Start, overwriting the cert file should NOT change the emitted data.
	// Certs are parsed only at Start time.
	dir := t.TempDir()
	path := filepath.Join(dir, "cert.pem")

	notAfter1 := time.Now().Add(10 * 24 * time.Hour).Truncate(time.Second)
	f1, err := os.Create(path)
	require.NoError(t, err)
	appendCertToFile(t, f1, notAfter1, "before.example.com", 1)
	require.NoError(t, f1.Close())

	cfg := &Config{
		InstanceID:           "test-instance",
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		CertFilePaths:        []string{path},
	}
	scraper := newTestScraper(t, cfg)

	metrics1, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	dp1 := metrics1.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter1.Unix(), dp1.IntValue())

	// Overwrite with a new cert (different expiry)
	notAfter2 := time.Now().Add(90 * 24 * time.Hour).Truncate(time.Second)
	f2, err := os.Create(path)
	require.NoError(t, err)
	appendCertToFile(t, f2, notAfter2, "after.example.com", 2)
	require.NoError(t, f2.Close())

	metrics2, err := scraper.Scrape(context.Background())
	require.NoError(t, err)
	dp2 := metrics2.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0)
	require.Equal(t, notAfter1.Unix(), dp2.IntValue(), "should still report original cert parsed at Start")
}
