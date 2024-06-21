// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxplusreceiver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"

	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginxinc/nginx-plus-go-client/client"
)

// var _ scraperhelper.Scraper = (*nginxPlusScraper)(nil)

const (
	plusAPIVersion = 9
)

type nginxPlusScraper struct {
	httpClient *http.Client
	plusClient *plusapi.NginxClient

	settings component.TelemetrySettings
	cfg      *Config
	mb       *metadata.MetricsBuilder
}

func newNginxPlusScraper(
	settings receiver.Settings,
	cfg *Config,
) (*nginxPlusScraper, error) {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)

	plusClient, err := plusapi.NewNginxClient(cfg.Endpoint,
		plusapi.WithAPIVersion(plusAPIVersion),
	)
	if err != nil {
		return nil, err
	}

	return &nginxPlusScraper{
		plusClient: plusClient,
		settings:   settings.TelemetrySettings,
		cfg:        cfg,
		mb:         mb,
	}, nil
}

// func (nps *nginxPlusScraper) ID() component.ID {
// 	return component.NewID(metadata.Type)
// }

// func (nps *nginxPlusScraper) Start(ctx context.Context, host component.Host) error {
// 	httpClient, err := nps.cfg.ToClient(ctx, host, nps.settings)
// 	if err != nil {
// 		return err
// 	}
// 	nps.httpClient = httpClient

// 	nps.plusClient

// 	return nil
// }

func (nps *nginxPlusScraper) scrape(context.Context) (pmetric.Metrics, error) {
	stats, err := nps.plusClient.GetStats()
	if err != nil {
		return pmetric.Metrics{}, fmt.Errorf("GET stats: %w", err)
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	nps.recordResponseStatuses(stats, now)
	return nps.mb.Emit(), nil
}

func (nps *nginxPlusScraper) recordResponseStatuses(stats *plusapi.Stats, now pcommon.Timestamp) {
	for lzName, lz := range stats.LocationZones {
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses2xx),
			metadata.AttributeNginxStatusRange2xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses3xx),
			metadata.AttributeNginxStatusRange3xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses4xx),
			metadata.AttributeNginxStatusRange4xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses5xx),
			metadata.AttributeNginxStatusRange5xx,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		nps.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, int64(lz.Discarded),
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
	}

	for szName, sz := range stats.ServerZones {
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses2xx),
			metadata.AttributeNginxStatusRange2xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses3xx),
			metadata.AttributeNginxStatusRange3xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses4xx),
			metadata.AttributeNginxStatusRange4xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses5xx),
			metadata.AttributeNginxStatusRange5xx,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, int64(sz.Discarded),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		nps.mb.RecordNginxHTTPRequestProcessingCountDataPoint(now, int64(sz.Processing),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
	}
}

// func (nps *nginxPlusScraper) Shutdown(ctx context.Context) error {
// 	return nil
// }
