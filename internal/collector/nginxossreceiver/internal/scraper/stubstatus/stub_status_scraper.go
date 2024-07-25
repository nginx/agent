// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stubstatus

import (
	"context"
	"net/http"
	"time"

	"github.com/nginxinc/nginx-prometheus-exporter/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
)

type NginxStubStatusScraper struct {
	httpClient *http.Client
	client     *client.NginxClient

	settings component.TelemetrySettings
	cfg      *config.Config
	mb       *metadata.MetricsBuilder
}

var _ scraperhelper.Scraper = (*NginxStubStatusScraper)(nil)

func NewScraper(
	settings receiver.Settings,
	cfg *config.Config,
) *NginxStubStatusScraper {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	return &NginxStubStatusScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       mb,
	}
}

func (s *NginxStubStatusScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (s *NginxStubStatusScraper) Start(ctx context.Context, host component.Host) error {
	httpClient, err := s.cfg.ToClient(ctx, host, s.settings)
	if err != nil {
		return err
	}
	s.httpClient = httpClient

	return nil
}

func (s *NginxStubStatusScraper) Shutdown(_ context.Context) error {
	return nil
}

func (s *NginxStubStatusScraper) Scrape(context.Context) (pmetric.Metrics, error) {
	// Init client in scrape method in case there are transient errors in the constructor.
	if s.client == nil {
		var err error
		s.client, err = client.NewNginxClient(s.httpClient, s.cfg.ClientConfig.Endpoint)
		if err != nil {
			s.client = nil
			return pmetric.Metrics{}, err
		}
	}

	stats, err := s.client.GetStubStats()
	if err != nil {
		s.settings.Logger.Error("fetch nginx stats", zap.Error(err))
		return pmetric.Metrics{}, err
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	s.mb.RecordNginxRequestsDataPoint(now, stats.Requests)
	s.mb.RecordNginxConnectionsAcceptedDataPoint(now, stats.Connections.Accepted)
	s.mb.RecordNginxConnectionsHandledDataPoint(now, stats.Connections.Handled)
	s.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Active, metadata.AttributeStateActive)
	s.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Reading, metadata.AttributeStateReading)
	s.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Writing, metadata.AttributeStateWriting)
	s.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Waiting, metadata.AttributeStateWaiting)

	return s.mb.Emit(), nil
}
