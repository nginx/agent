// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver/internal/metadata"
	plusapi "github.com/nginx/nginx-plus-go-client/v3/client"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type ServerZoneMetrics struct {
	PreviousServerZoneResponses map[string]ResponseStatuses
	PreviousServerZoneRequests  map[string]int64
	mb                          *metadata.MetricsBuilder
}

func NewServerZoneMetrics(stats *plusapi.Stats, mb *metadata.MetricsBuilder) *ServerZoneMetrics {
	return &ServerZoneMetrics{
		mb:                          mb,
		PreviousServerZoneResponses: createPreviousServerZoneResponses(stats),
		PreviousServerZoneRequests:  createPreviousServerZoneRequests(stats),
	}
}

func (szm *ServerZoneMetrics) RecordServerZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for szName, sz := range stats.ServerZones {
		szm.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			int64(sz.Received),
			metadata.AttributeNginxIoDirectionReceive,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		szm.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			int64(sz.Sent),
			metadata.AttributeNginxIoDirectionTransmit,
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		szm.mb.RecordNginxHTTPRequestsDataPoint(now, int64(sz.Requests), szName, metadata.AttributeNginxZoneTypeSERVER)

		szm.mb.RecordNginxHTTPRequestCountDataPoint(now,
			int64(sz.Requests)-szm.PreviousServerZoneRequests[szName],
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)
		szm.PreviousServerZoneRequests[szName] = int64(sz.Requests)

		szm.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, int64(sz.Discarded),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		szm.mb.RecordNginxHTTPRequestProcessingCountDataPoint(now, int64(sz.Processing),
			szName,
			metadata.AttributeNginxZoneTypeSERVER,
		)

		szm.recordServerZoneHTTPMetrics(sz, szName, now)
	}
}

//nolint:dupl // Duplicate of recordLocationZoneHTTPMetrics but same function can not be used due to plusapi.ServerZone
func (szm *ServerZoneMetrics) recordServerZoneHTTPMetrics(sz plusapi.ServerZone, szName string, now pcommon.Timestamp) {
	// Response Status
	szm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses1xx),
		metadata.AttributeNginxStatusRange1xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER,
	)
	szm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses2xx),
		metadata.AttributeNginxStatusRange2xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER,
	)
	szm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses3xx),
		metadata.AttributeNginxStatusRange3xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER,
	)

	szm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses4xx),
		metadata.AttributeNginxStatusRange4xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER,
	)

	szm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(sz.Responses.Responses5xx),
		metadata.AttributeNginxStatusRange5xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER,
	)

	// Response Count Status
	szm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses1xx)-szm.PreviousServerZoneResponses[szName].OneHundredStatusRange,
		metadata.AttributeNginxStatusRange1xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	szm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses2xx)-szm.PreviousServerZoneResponses[szName].TwoHundredStatusRange,
		metadata.AttributeNginxStatusRange2xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	szm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses3xx)-szm.PreviousServerZoneResponses[szName].ThreeHundredStatusRange,
		metadata.AttributeNginxStatusRange3xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	szm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses4xx)-szm.PreviousServerZoneResponses[szName].FourHundredStatusRange,
		metadata.AttributeNginxStatusRange4xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	szm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(sz.Responses.Responses5xx)-szm.PreviousServerZoneResponses[szName].FiveHundredStatusRange,
		metadata.AttributeNginxStatusRange5xx,
		szName,
		metadata.AttributeNginxZoneTypeSERVER)

	respStatus := ResponseStatuses{
		OneHundredStatusRange:   int64(sz.Responses.Responses1xx),
		TwoHundredStatusRange:   int64(sz.Responses.Responses2xx),
		ThreeHundredStatusRange: int64(sz.Responses.Responses3xx),
		FourHundredStatusRange:  int64(sz.Responses.Responses4xx),
		FiveHundredStatusRange:  int64(sz.Responses.Responses5xx),
	}

	szm.PreviousServerZoneResponses[szName] = respStatus
}

func createPreviousServerZoneRequests(stats *plusapi.Stats) map[string]int64 {
	previousServerZoneRequests := make(map[string]int64)
	for szName, sz := range stats.ServerZones {
		previousServerZoneRequests[szName] = int64(sz.Requests)
	}

	return previousServerZoneRequests
}

func createPreviousServerZoneResponses(stats *plusapi.Stats) map[string]ResponseStatuses {
	previousServerZoneResponses := make(map[string]ResponseStatuses)
	for szName, sz := range stats.ServerZones {
		respStatus := ResponseStatuses{
			OneHundredStatusRange:   int64(sz.Responses.Responses1xx),
			TwoHundredStatusRange:   int64(sz.Responses.Responses2xx),
			ThreeHundredStatusRange: int64(sz.Responses.Responses3xx),
			FourHundredStatusRange:  int64(sz.Responses.Responses4xx),
			FiveHundredStatusRange:  int64(sz.Responses.Responses5xx),
		}

		previousServerZoneResponses[szName] = respStatus
	}

	return previousServerZoneResponses
}
