// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// nolint: lll
const (
	accessLogPattern = `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for"`
	accessLogLine    = "127.0.0.1 - - [19/May/2022:09:30:39 +0000] \"GET /nginx_status HTTP/1.1\" 200 98 \"-\" \"Go-http-client/1.1\" \"-\"\n"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "access_log_file_input", config.OperatorID)
}

func TestConfig_Build(t *testing.T) {
	tempFile, err := os.CreateTemp(os.TempDir(), "access.log")
	require.NoError(t, err)
	defer tempFile.Close()

	config := NewConfig()
	config.Include = []string{tempFile.Name()}

	telemetrySettings := component.TelemetrySettings{
		Logger:        newLogger(t),
		MeterProvider: metricSdk.NewMeterProvider(),
	}

	operator, err := config.Build(telemetrySettings)

	require.NoError(t, err)
	assert.NotNil(t, operator)
	assert.Equal(t, "access_log_file_input", operator.Type())
}

func Test_newNginxAccessItem(t *testing.T) {
	item, err := newNginxAccessItem(
		map[string]string{
			"status": "200",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "200", item.Status)
}

func Test_grokParseFunction(t *testing.T) {
	logger := zap.L()
	grok, err := NewCompiledGrok(accessLogPattern, logger)
	require.NoError(t, err)

	token := []byte(accessLogLine)

	function := grokParseFunction(logger, grok)

	result := function(token)
	item, ok := result.(*model.NginxAccessItem)
	assert.True(t, ok)
	assert.Equal(t, "200", item.Status)
}

func Test_copyFunction(t *testing.T) {
	logger := zap.L()
	grok, err := NewCompiledGrok(accessLogPattern, logger)
	require.NoError(t, err)

	token := []byte(accessLogLine)

	function := copyFunction(logger, grok)

	result := function(token)
	item, ok := result.(*model.NginxAccessItem)
	assert.True(t, ok)
	assert.Equal(t, "200", item.Status)
}
