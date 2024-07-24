// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package record

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const testDataDir = "testdata"

func TestRecordAccessItem(t *testing.T) {
	tests := []struct {
		name         string
		input        []*model.NginxAccessItem
		expectedPath string
		shouldErr    bool
		expErrMsg    string
	}{
		{
			name: "basic nginx.http.status case",
			input: []*model.NginxAccessItem{
				{
					BodyBytesSent:          "615",
					Status:                 "200",
					RemoteAddress:          "127.0.0.1",
					HTTPUserAgent:          "PostmanRuntime/7.36.1",
					Request:                "GET / HTTP/1.1",
					BytesSent:              "853",
					RequestLength:          "226",
					RequestTime:            "0.000",
					GzipRatio:              "-",
					ServerProtocol:         "HTTP/1.1",
					UpstreamConnectTime:    "-",
					UpstreamHeaderTime:     "-",
					UpstreamResponseTime:   "-",
					UpstreamResponseLength: "-",
					UpstreamStatus:         "",
					UpstreamCacheStatus:    "",
				},
				{
					BodyBytesSent:          "28",
					Status:                 "200",
					RemoteAddress:          "127.0.0.1",
					HTTPUserAgent:          "PostmanRuntime/7.36.1",
					Request:                "GET /frontend1 HTTP/1.1",
					BytesSent:              "190",
					RequestLength:          "235",
					RequestTime:            "0.004",
					GzipRatio:              "-",
					ServerProtocol:         "HTTP/1.1",
					UpstreamConnectTime:    "0.003",
					UpstreamHeaderTime:     "0.003",
					UpstreamResponseTime:   "0.003",
					UpstreamResponseLength: "28",
					UpstreamStatus:         "",
					UpstreamCacheStatus:    "",
				},
			},
			expectedPath: "basic-nginx.http.status.yaml",
		},
		{
			name: "all nginx.http.status status codes",
			input: []*model.NginxAccessItem{
				{ // The recorder only parses the status code for nginx.http.status, omitting other fields for brevity.
					Status: "100",
				},
				{
					Status: "103",
				},
				{
					Status: "200",
				},
				{
					Status: "202",
				},
				{
					Status: "300",
				},
				{
					Status: "302",
				},
				{
					Status: "400",
				},
				{
					Status: "404",
				},
				{
					Status: "500",
				},
				{
					Status: "502",
				},
			},
			expectedPath: "multicode-nginx.http.status.yaml",
		},
		{
			name: "random string in status code",
			input: []*model.NginxAccessItem{
				{
					Status: "not-a-status-code",
				},
			},
			shouldErr: true,
			expErrMsg: "cast status code to int",
		},
		{
			name: "non-existent status code range",
			input: []*model.NginxAccessItem{
				{
					Status: "700",
				},
			},
			shouldErr: true,
			expErrMsg: "unknown code range: 700",
		},
	}

	mb := metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), receivertest.NewNopCreateSettings())

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			var err error
			for _, item := range test.input {
				recordErr := Item(item, mb)
				err = errors.Join(err, recordErr)
			}

			if test.shouldErr {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), test.expErrMsg)
			} else {
				require.NoError(tt, err)
				expectedFile := filepath.Join(testDataDir, test.expectedPath)
				expected, err := golden.ReadMetrics(expectedFile)
				require.NoError(t, err)

				actual := mb.Emit()
				require.NoError(tt, pmetrictest.CompareMetrics(expected, actual,
					pmetrictest.IgnoreStartTimestamp(),
					pmetrictest.IgnoreMetricDataPointsOrder(),
					pmetrictest.IgnoreTimestamp(),
					pmetrictest.IgnoreMetricsOrder()))
			}
		})
	}
}
