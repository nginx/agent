// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package certificatereceiver

import (
	"context"
	"time"

	"github.com/nginx/agent/v3/internal/collector/certificatereceiver/internal/metadata"
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
	}
}

func (c *CertificateScraper) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (c *CertificateScraper) Scrape(_ context.Context) (pmetric.Metrics, error) {
	if len(c.cfg.CertMeta) == 0 {
		return pmetric.NewMetrics(), nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	if c.cfg.InstanceID != "" {
		c.rb.SetInstanceID(c.cfg.InstanceID)
	}

	for path, info := range c.cfg.CertMeta {
		c.mb.RecordNginxSslCertificateExpiryDataPoint(
			now, info.NotAfter,
			path,
			info.PublicKeyAlgorithm,
			info.SerialNumber,
			info.CommonName,
		)
	}

	return c.mb.Emit(metadata.WithResource(c.rb.Emit())), nil
}

func (c *CertificateScraper) Shutdown(_ context.Context) error {
	return nil
}
