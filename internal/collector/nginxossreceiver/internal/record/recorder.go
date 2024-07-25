// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package record

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	// Needed for "magic number" linter.
	status100 = 100
	status200 = 200
	status300 = 300
	status400 = 400
	status500 = 500

	percent = 100
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
	codeRange := (number / percent) * percent

	switch codeRange {
	case status100:
		return metadata.AttributeStatusCode1xx, nil
	case status200:
		return metadata.AttributeStatusCode2xx, nil
	case status300:
		return metadata.AttributeStatusCode3xx, nil
	case status400:
		return metadata.AttributeStatusCode4xx, nil
	case status500:
		return metadata.AttributeStatusCode5xx, nil
	default:
		return 0, fmt.Errorf("unknown code range: %d", codeRange)
	}
}
