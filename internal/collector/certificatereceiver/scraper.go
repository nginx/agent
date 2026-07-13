// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type certCacheEntry struct {
	mtime time.Time
	certs []*x509.Certificate
}

type CertificateScraper struct {
	cfg    *Config
	mb     *metadata.MetricsBuilder
	rb     *metadata.ResourceBuilder
	logger *zap.Logger
	cache  map[string]*certCacheEntry
}

func newCertificateScraper(
	settings receiver.Settings,
	cfg *Config,
) *CertificateScraper {
	logger := settings.Logger
	logger.Info("Creating certificate scraper")
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &CertificateScraper{
		cfg:    cfg,
		mb:     mb,
		rb:     rb,
		logger: logger,
		cache:  make(map[string]*certCacheEntry),
	}
}

func (c *CertificateScraper) Start(ctx context.Context, _ component.Host) error {
	for _, path := range c.cfg.CertFilePaths {
		if err := c.refreshCacheEntry(path); err != nil {
			slog.WarnContext(ctx, "Failed to pre-load certificate file", "path", path, "error", err)
		}
	}

	return nil
}

func (c *CertificateScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	if len(c.cfg.CertFilePaths) == 0 {
		return pmetric.NewMetrics(), nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	if c.cfg.InstanceID != "" {
		c.rb.SetInstanceID(c.cfg.InstanceID)
	}

	const hexBase = 16

	for _, path := range c.cfg.CertFilePaths {
		if err := c.refreshCacheEntry(path); err != nil {
			slog.WarnContext(ctx, "Failed to load certificate file, skipping", "path", path, "error", err)

			continue
		}

		for _, cert := range c.cache[path].certs {
			c.mb.RecordNginxCertificateExpiryDataPoint(
				now, cert.NotAfter.Unix(),
				path,
				cert.PublicKeyAlgorithm.String(),
				cert.SerialNumber.Text(hexBase),
				cert.Subject.CommonName,
			)
		}
	}

	return c.mb.Emit(metadata.WithResource(c.rb.Emit())), nil
}

func (c *CertificateScraper) Shutdown(_ context.Context) error {
	return nil
}

// refreshCacheEntry stats the file and re-parses it only if the mtime has changed.
// On any error the cache entry is cleared so the next call retries.
func (c *CertificateScraper) refreshCacheEntry(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		delete(c.cache, path)

		return fmt.Errorf("stat %s: %w", path, err)
	}

	if entry, ok := c.cache[path]; ok && entry.mtime.Equal(info.ModTime()) {
		return nil
	}

	certs, err := parseCertFile(path)
	if err != nil {
		delete(c.cache, path)

		return err
	}

	c.cache[path] = &certCacheEntry{
		mtime: info.ModTime(),
		certs: certs,
	}

	return nil
}

func parseCertFile(path string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var certs []*x509.Certificate

	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)

		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("parse certificate in %s: %w", path, parseErr)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in %s", path)
	}

	return certs, nil
}
