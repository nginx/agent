// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
)

type nginxPlusScraper struct {
	httpClient *http.Client

	settings component.TelemetrySettings
	cfg      *Config
	mb       *metadata.MetricsBuilder
}

func newNginxPlusScraper(
	settings receiver.Settings,
	cfg *Config,
) *nginxPlusScraper {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	return &nginxPlusScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       mb,
	}
}

func (r *nginxPlusScraper) start(ctx context.Context, host component.Host) error {
	httpClient, err := r.cfg.ToClient(ctx, host, r.settings)
	if err != nil {
		return err
	}
	r.httpClient = httpClient

	return nil
}

func (r *nginxPlusScraper) scrape(context.Context) (pmetric.Metrics, error) {
	// Init client in scrape method in case there are transient errors in the constructor.
	// if r.client == nil {
	// 	var err error
	// 	r.client, err = client.NewNginxClient(r.httpClient, r.cfg.ClientConfig.Endpoint)
	// 	if err != nil {
	// 		r.client = nil
	// 		return pmetric.Metrics{}, err
	// 	}
	// }

	// stats, err := r.client.GetStubStats()
	// if err != nil {
	// 	r.settings.Logger.Error("Failed to fetch nginx stats", zap.Error(err))
	// 	return pmetric.Metrics{}, err
	// }

	// now := pcommon.NewTimestampFromTime(time.Now())
	// r.mb.RecordNginxRequestsDataPoint(now, stats.Requests)
	// r.mb.RecordNginxConnectionsAcceptedDataPoint(now, stats.Connections.Accepted)
	// r.mb.RecordNginxConnectionsHandledDataPoint(now, stats.Connections.Handled)
	// r.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Active, metadata.AttributeStateActive)
	// r.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Reading, metadata.AttributeStateReading)
	// r.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Writing, metadata.AttributeStateWriting)
	// r.mb.RecordNginxConnectionsCurrentDataPoint(now, stats.Connections.Waiting, metadata.AttributeStateWaiting)
	return pmetric.Metrics{}, nil
}
