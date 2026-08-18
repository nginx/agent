// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"crypto/x509"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/datasource/cert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type CertificateScraper struct {
	cfg    *Config
	mb     *metadata.MetricsBuilder
	rb     *metadata.ResourceBuilder
	logger *zap.Logger
	cache  map[string]*x509.Certificate
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
		cache:  make(map[string]*x509.Certificate),
	}
}

func (c *CertificateScraper) Start(ctx context.Context, _ component.Host) error {
	for _, path := range c.cfg.CertFilePaths {
		loadedCert, err := cert.LoadCertificate(path)
		if err != nil {
			slog.WarnContext(ctx, "Failed to load certificate file", "path", path, "error", err)

			continue
		}

		c.cache[path] = loadedCert
	}

	return nil
}

func (c *CertificateScraper) Scrape(_ context.Context) (pmetric.Metrics, error) {
	if len(c.cfg.CertFilePaths) == 0 {
		return pmetric.NewMetrics(), nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	if c.cfg.InstanceID != "" {
		c.rb.SetInstanceID(c.cfg.InstanceID)
	}

	for _, path := range c.cfg.CertFilePaths {
		loadedCert, ok := c.cache[path]
		if !ok {
			continue
		}

		c.mb.RecordNginxSslCertificateExpiryDataPoint(
			now, loadedCert.NotAfter.Unix(),
			path,
			loadedCert.PublicKeyAlgorithm.String(),
			loadedCert.SerialNumber.String(),
			loadedCert.Subject.CommonName,
		)
	}

	return c.mb.Emit(metadata.WithResource(c.rb.Emit())), nil
}

func (c *CertificateScraper) Shutdown(_ context.Context) error {
	return nil
}
