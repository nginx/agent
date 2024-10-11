// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stubstatus

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/nginxinc/nginx-prometheus-exporter/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
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
	resource *resource.Resource
}

var _ scraperhelper.Scraper = (*NginxStubStatusScraper)(nil)

func NewScraper(
	settings receiver.Settings,
	cfg *config.Config,
) *NginxStubStatusScraper {
	logger := settings.Logger
	logger.Info("Creating NGINX stub status scraper")

	instanceNameType := strings.Split(settings.ID.Name(), "/") // nginxplus/1c3245-...
	stubResource, err := resource.New(
		context.Background(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String("ngx.instance.type", "nginx"),
			attribute.String("ngx.instance.id", instanceNameType[0]),
		),
	)
	if err != nil {
		return nil
	}

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	settings.Logger.Info("N+ scraper: ",
		zap.Any("type", "nginx"),
		zap.Any("id", instanceNameType[0]),
		zap.Any("stubResource", stubResource))

	return &NginxStubStatusScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       mb,
		resource: stubResource,
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
		s.client = client.NewNginxClient(s.httpClient, s.cfg.ClientConfig.Endpoint)
	}

	stats, err := s.client.GetStubStats()
	if err != nil {
		s.settings.Logger.Error("fetch nginx stats", zap.Error(err))
		return pmetric.Metrics{}, err
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	s.mb.RecordNginxHTTPRequestsDataPoint(now, stats.Requests)

	s.mb.RecordNginxHTTPConnDataPoint(
		now,
		stats.Connections.Accepted,
		metadata.AttributeNginxConnOutcomeACCEPTED,
	)
	s.mb.RecordNginxHTTPConnDataPoint(
		now,
		stats.Connections.Handled,
		metadata.AttributeNginxConnOutcomeHANDLED,
	)

	s.mb.RecordNginxHTTPConnCountDataPoint(
		now,
		stats.Connections.Active,
		metadata.AttributeNginxConnOutcomeACTIVE,
	)
	s.mb.RecordNginxHTTPConnCountDataPoint(
		now,
		stats.Connections.Reading,
		metadata.AttributeNginxConnOutcomeREADING,
	)
	s.mb.RecordNginxHTTPConnCountDataPoint(
		now,
		stats.Connections.Writing,
		metadata.AttributeNginxConnOutcomeWRITING,
	)
	s.mb.RecordNginxHTTPConnCountDataPoint(
		now,
		stats.Connections.Waiting,
		metadata.AttributeNginxConnOutcomeWAITING,
	)

	return s.mb.Emit(), nil
}
