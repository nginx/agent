// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stubstatus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nginxinc/nginx-prometheus-exporter/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
)

type NginxStubStatusScraper struct {
	logger           *zap.Logger
	httpClient       *http.Client
	client           *client.NginxClient
	cfg              *config.Config
	mb               *metadata.MetricsBuilder
	rb               *metadata.ResourceBuilder
	settings         receiver.Settings
	init             sync.Once
	previousRequests int
}

func NewScraper(
	settings receiver.Settings,
	cfg *config.Config,
) *NginxStubStatusScraper {
	logger := settings.Logger
	logger.Info("Creating NGINX stub status scraper")

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &NginxStubStatusScraper{
		settings: settings,
		cfg:      cfg,
		mb:       mb,
		rb:       rb,
		logger:   logger,
	}
}

func (s *NginxStubStatusScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

// nolint: unparam
func (s *NginxStubStatusScraper) Start(_ context.Context, _ component.Host) error {
	s.logger.Info("Starting NGINX stub status scraper")
	httpClient := http.DefaultClient
	caCertLocation := s.cfg.APIDetails.Ca
	if caCertLocation != "" {
		s.settings.Logger.Debug("Reading CA certificate", zap.Any("file_path", caCertLocation))
		caCert, err := os.ReadFile(caCertLocation)
		if err != nil {
			s.settings.Logger.Error("Error starting NGINX stub status scraper. "+
				"Failed to read CA certificate", zap.Error(err))

			return nil
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    caCertPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}
	}
	httpClient.Timeout = s.cfg.ClientConfig.Timeout

	if strings.HasPrefix(s.cfg.APIDetails.Listen, "unix:") {
		httpClient.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", strings.TrimPrefix(s.cfg.APIDetails.Listen, "unix:"))
			},
		}
	}
	s.httpClient = httpClient

	return nil
}

// nolint: unparam
func (s *NginxStubStatusScraper) Shutdown(_ context.Context) error {
	s.logger.Info("Shutting down NGINX stub status scraper")
	return nil
}

func (s *NginxStubStatusScraper) Scrape(context.Context) (pmetric.Metrics, error) {
	// s.init.Do is ran only once, it is only ran the first time Scrape is called to set the previous requests
	// metric value
	s.init.Do(func() {
		// Init client in scrape method in case there are transient errors in the constructor.
		if s.client == nil {
			s.client = client.NewNginxClient(s.httpClient, s.cfg.APIDetails.URL)
		}

		stats, err := s.client.GetStubStats()
		if err != nil {
			s.settings.Logger.Error("Failed to get stats from stub status API", zap.Error(err))
			return
		}

		s.previousRequests = int(stats.Requests)
	})

	// Init client in scrape method in case there are transient errors in the constructor.
	if s.client == nil {
		s.client = client.NewNginxClient(s.httpClient, s.cfg.APIDetails.URL)
	}

	stats, err := s.client.GetStubStats()
	if err != nil {
		s.settings.Logger.Error("Failed to get stats from stub status API", zap.Error(err))
		return pmetric.Metrics{}, err
	}

	s.rb.SetInstanceID(s.settings.ID.Name())
	s.rb.SetInstanceType("nginx")
	s.settings.Logger.Debug("NGINX OSS stub status resource info", zap.Any("resource", s.rb))

	now := pcommon.NewTimestampFromTime(time.Now())

	s.mb.RecordNginxHTTPRequestsDataPoint(now, stats.Requests)

	s.mb.RecordNginxHTTPRequestCountDataPoint(now, int64(int(stats.Requests)-s.previousRequests))
	s.previousRequests = int(stats.Requests)

	s.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		stats.Connections.Accepted,
		metadata.AttributeNginxConnectionsOutcomeACCEPTED,
	)
	s.mb.RecordNginxHTTPConnectionsDataPoint(
		now,
		stats.Connections.Handled,
		metadata.AttributeNginxConnectionsOutcomeHANDLED,
	)

	s.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		stats.Connections.Active,
		metadata.AttributeNginxConnectionsOutcomeACTIVE,
	)
	s.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		stats.Connections.Reading,
		metadata.AttributeNginxConnectionsOutcomeREADING,
	)
	s.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		stats.Connections.Writing,
		metadata.AttributeNginxConnectionsOutcomeWRITING,
	)
	s.mb.RecordNginxHTTPConnectionCountDataPoint(
		now,
		stats.Connections.Waiting,
		metadata.AttributeNginxConnectionsOutcomeWAITING,
	)

	return s.mb.Emit(metadata.WithResource(s.rb.Emit())), nil
}
