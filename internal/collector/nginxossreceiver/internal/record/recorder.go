// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package record

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Item extracts data from NGINX Access Items and records them using the given MetricsBuilder.
func Item(ai *model.NginxAccessItem, mb *metadata.MetricsBuilder) error {
	now := pcommon.NewTimestampFromTime(time.Now())

	if ai.Status != "" {
		codeRange, err := mapCodeRange(ai.Status)
		if err != nil {
			return fmt.Errorf("code range parse: %w", err)
		}

		mb.RecordNginxHTTPStatusDataPoint(now, 1, codeRange)
	}

	return nil
}

func mapCodeRange(statusCode string) (metadata.AttributeStatusCode, error) {
	number, err := strconv.Atoi(statusCode)
	if err != nil {
		return 0, fmt.Errorf("cast status code to int: %w", err)
	}

	// We want to "floor" the response code, so we can map it to the correct range (i.e. to 1xx, 2xx, 4xx or 5xx).
	codeRange := (number / 100) * 100

	switch codeRange {
	case 100:
		return metadata.AttributeStatusCode1xx, nil
	case 200:
		return metadata.AttributeStatusCode2xx, nil
	case 300:
		return metadata.AttributeStatusCode3xx, nil
	case 400:
		return metadata.AttributeStatusCode4xx, nil
	case 500:
		return metadata.AttributeStatusCode5xx, nil
	default:
		return 0, fmt.Errorf("unknown code range: %d", codeRange)
	}
}
