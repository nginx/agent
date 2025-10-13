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

type LocationZoneMetrics struct {
	PreviousLocationZoneResponses map[string]ResponseStatuses
	PreviousLocationZoneRequests  map[string]int64
	mb                            *metadata.MetricsBuilder
}

type ResponseStatuses struct {
	OneHundredStatusRange   int64
	TwoHundredStatusRange   int64
	ThreeHundredStatusRange int64
	FourHundredStatusRange  int64
	FiveHundredStatusRange  int64
}

func NewLocationZoneMetrics(stats *plusapi.Stats, mb *metadata.MetricsBuilder) *LocationZoneMetrics {
	return &LocationZoneMetrics{
		mb:                            mb,
		PreviousLocationZoneResponses: createPreviousLocationZoneResponses(stats),
		PreviousLocationZoneRequests:  createPreviousLocationZoneRequests(stats),
	}
}

func (lzm *LocationZoneMetrics) RecordLocationZoneMetrics(stats *plusapi.Stats, now pcommon.Timestamp) {
	for lzName, lz := range stats.LocationZones {
		// Requests
		lzm.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			lz.Received,
			metadata.AttributeNginxIoDirectionReceive,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)
		lzm.mb.RecordNginxHTTPRequestIoDataPoint(
			now,
			lz.Sent,
			metadata.AttributeNginxIoDirectionTransmit,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		lzm.mb.RecordNginxHTTPRequestsDataPoint(
			now,
			lz.Requests,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		lzm.mb.RecordNginxHTTPRequestCountDataPoint(now,
			lz.Requests-lzm.PreviousLocationZoneRequests[lzName],
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		lzm.mb.RecordNginxHTTPRequestDiscardedDataPoint(now, lz.Discarded,
			lzName,
			metadata.AttributeNginxZoneTypeLOCATION,
		)

		lzm.PreviousLocationZoneRequests[lzName] = lz.Requests

		lzm.recordLocationZoneHTTPMetrics(lz, lzName, now)
	}
}

//nolint:dupl // Duplicate of recordServerZoneHTTPMetrics but same function can not be used due to plusapi.LocationZone
func (lzm *LocationZoneMetrics) recordLocationZoneHTTPMetrics(lz plusapi.LocationZone,
	lzName string, now pcommon.Timestamp,
) {
	// Response Status
	lzm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses1xx),
		metadata.AttributeNginxStatusRange1xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION,
	)
	lzm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses2xx),
		metadata.AttributeNginxStatusRange2xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION,
	)
	lzm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses3xx),
		metadata.AttributeNginxStatusRange3xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION,
	)

	lzm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses4xx),
		metadata.AttributeNginxStatusRange4xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION,
	)

	lzm.mb.RecordNginxHTTPResponseStatusDataPoint(now, int64(lz.Responses.Responses5xx),
		metadata.AttributeNginxStatusRange5xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION,
	)

	// Requests
	lzm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses1xx)-lzm.PreviousLocationZoneResponses[lzName].OneHundredStatusRange,
		metadata.AttributeNginxStatusRange1xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	// Response Count Status
	lzm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses2xx)-lzm.PreviousLocationZoneResponses[lzName].TwoHundredStatusRange,
		metadata.AttributeNginxStatusRange2xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	lzm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses3xx)-lzm.PreviousLocationZoneResponses[lzName].ThreeHundredStatusRange,
		metadata.AttributeNginxStatusRange3xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	lzm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses4xx)-lzm.PreviousLocationZoneResponses[lzName].FourHundredStatusRange,
		metadata.AttributeNginxStatusRange4xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	lzm.mb.RecordNginxHTTPResponseCountDataPoint(now,
		int64(lz.Responses.Responses5xx)-lzm.PreviousLocationZoneResponses[lzName].FiveHundredStatusRange,
		metadata.AttributeNginxStatusRange5xx,
		lzName,
		metadata.AttributeNginxZoneTypeLOCATION)

	respStatus := ResponseStatuses{
		OneHundredStatusRange:   int64(lz.Responses.Responses1xx),
		TwoHundredStatusRange:   int64(lz.Responses.Responses2xx),
		ThreeHundredStatusRange: int64(lz.Responses.Responses3xx),
		FourHundredStatusRange:  int64(lz.Responses.Responses4xx),
		FiveHundredStatusRange:  int64(lz.Responses.Responses5xx),
	}

	lzm.PreviousLocationZoneResponses[lzName] = respStatus
}

func createPreviousLocationZoneResponses(stats *plusapi.Stats) map[string]ResponseStatuses {
	previousLocationZoneResponses := make(map[string]ResponseStatuses)
	for lzName, lz := range stats.LocationZones {
		respStatus := ResponseStatuses{
			OneHundredStatusRange:   int64(lz.Responses.Responses1xx),
			TwoHundredStatusRange:   int64(lz.Responses.Responses2xx),
			ThreeHundredStatusRange: int64(lz.Responses.Responses3xx),
			FourHundredStatusRange:  int64(lz.Responses.Responses4xx),
			FiveHundredStatusRange:  int64(lz.Responses.Responses5xx),
		}

		previousLocationZoneResponses[lzName] = respStatus
	}

	return previousLocationZoneResponses
}

func createPreviousLocationZoneRequests(stats *plusapi.Stats) map[string]int64 {
	previousLocationZoneRequests := make(map[string]int64)
	for lzName, lz := range stats.LocationZones {
		previousLocationZoneRequests[lzName] = lz.Requests
	}

	return previousLocationZoneRequests
}
