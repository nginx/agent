// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginxplusreceiver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/record"
	"go.opentelemetry.io/collector/component"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginxinc/nginx-plus-go-client/v2/client"
)

type NginxPlusScraper struct {
	locationZoneMetrics *record.LocationZoneMetrics
	serverZoneMetrics   *record.ServerZoneMetrics
	httpMetrics         *record.HTTPMetrics
	plusClient          *plusapi.NginxClient
	cfg                 *Config
	mb                  *metadata.MetricsBuilder
	rb                  *metadata.ResourceBuilder
	logger              *zap.Logger
	settings            receiver.Settings
	init                sync.Once
}

func newNginxPlusScraper(
	settings receiver.Settings,
	cfg *Config,
) *NginxPlusScraper {
	logger := settings.Logger
	logger.Info("Creating NGINX Plus scraper")
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &NginxPlusScraper{
		settings: settings,
		cfg:      cfg,
		mb:       mb,
		rb:       rb,
		logger:   settings.Logger,
	}
}

func (nps *NginxPlusScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (nps *NginxPlusScraper) Start(ctx context.Context, _ component.Host) error {
	endpoint := strings.TrimPrefix(nps.cfg.APIDetails.URL, "unix:")
	httpClient := http.DefaultClient
	caCertLocation := nps.cfg.APIDetails.Ca
	if caCertLocation != "" {
		nps.logger.Debug("Reading CA certificate", zap.Any("file_path", caCertLocation))
		caCert, err := os.ReadFile(caCertLocation)
		if err != nil {
			nps.logger.Error("Error starting NGINX stub status scraper. "+
				"Failed to read CA certificate", zap.Error(err))

			return err
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
	httpClient.Timeout = nps.cfg.ClientConfig.Timeout

	if strings.HasPrefix(nps.cfg.APIDetails.Listen, "unix:") {
		httpClient = socketClient(ctx, strings.TrimPrefix(nps.cfg.APIDetails.Listen, "unix:"))
	}

	plusClient, err := plusapi.NewNginxClient(endpoint,
		plusapi.WithMaxAPIVersion(), plusapi.WithHTTPClient(httpClient),
	)
	nps.plusClient = plusClient
	if err != nil {
		return err
	}

	return nil
}

func (nps *NginxPlusScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// nps.init.Do is ran only once, it is only ran the first time scrape is called to set the previous responses
	// metric value
	nps.init.Do(func() {
		stats, err := nps.plusClient.GetStats(ctx)
		if err != nil {
			nps.logger.Error("Failed to get stats from plus API", zap.Error(err))
			return
		}

		nps.httpMetrics = record.NewHTTPMetrics(stats, nps.mb)
		nps.locationZoneMetrics = record.NewLocationZoneMetrics(stats, nps.mb)
		nps.serverZoneMetrics = record.NewServerZoneMetrics(stats, nps.mb)
	})

	stats, err := nps.plusClient.GetStats(ctx)
	if err != nil {
		return pmetric.Metrics{}, fmt.Errorf("failed to get stats from plus API: %w", err)
	}

	nps.rb.SetInstanceID(nps.cfg.InstanceID)
	nps.rb.SetInstanceType("nginxplus")
	nps.logger.Debug("NGINX Plus resource info", zap.Any("resource", nps.rb))

	nps.logger.Debug("NGINX Plus stats", zap.Any("stats", stats))
	nps.recordMetrics(stats)

	return nps.mb.Emit(metadata.WithResource(nps.rb.Emit())), nil
}

func (nps *NginxPlusScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (nps *NginxPlusScraper) recordMetrics(stats *plusapi.Stats) {
	now := pcommon.NewTimestampFromTime(time.Now())

	// NGINX config reloads
	nps.mb.RecordNginxConfigReloadsDataPoint(now, int64(stats.NginxInfo.Generation))

	nps.httpMetrics.RecordHTTPMetrics(stats, now)
	nps.httpMetrics.RecordHTTPLimitMetrics(stats, now)

	nps.locationZoneMetrics.RecordLocationZoneMetrics(stats, now)
	nps.serverZoneMetrics.RecordServerZoneMetrics(stats, now)

	record.RecordCacheMetrics(nps.mb, stats, now)

	record.RecordHTTPUpstreamPeerMetrics(nps.mb, stats, now)
	record.RecordStreamMetrics(nps.mb, stats, now)

	record.RecordSlabPageMetrics(nps.mb, stats, now, nps.logger)
	record.RecordSSLMetrics(nps.mb, now, stats)
}

func socketClient(ctx context.Context, socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}
