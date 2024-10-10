// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const baseformat = `$remote_addr - $remote_user [$time_local] "$request"` +
	` $status $body_bytes_sent "$http_referer" "$http_user_agent"` +
	` "$http_x_forwarded_for" "$bytes_sent" "$request_length" "$request_time"` +
	` "$gzip_ratio" "$server_protocol" "$upstream_connect_time""$upstream_header_time"` +
	` "$upstream_response_length" "$upstream_response_time"`

func TestGrok_Constructor(t *testing.T) {
	logger := newLogger(t)

	tests := []struct {
		logger    *zap.Logger
		name      string
		logFormat string
		expErrMsg string
		shouldErr bool
	}{
		{
			name:      "Test 1: logger is nil",
			logger:    nil,
			shouldErr: true,
			expErrMsg: "logger cannot be nil",
		},
		{
			name:      "Test 2: valid log format",
			logFormat: baseformat,
			logger:    logger,
			shouldErr: false,
		},
		{
			name:      "Test 3: empty log format",
			logFormat: "",
			logger:    logger,
			shouldErr: false,
		},
		{
			name:      "Test 4: unknown log variable",
			logFormat: `$remote_addr - $remote_user [$time_local] "$unknown_variable" `,
			logger:    logger,
			shouldErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			grok, err := NewCompiledGrok(test.logFormat, test.logger)
			if test.shouldErr {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), test.expErrMsg)
			} else {
				require.NoError(tt, err)
				require.NotNil(tt, grok)
			}
		})
	}
}

func TestGrok_ParseString(t *testing.T) {
	logger := newLogger(t)

	tests := []struct {
		expOutput     map[string]string
		name          string
		logFormat     string
		inputLogEntry string
	}{
		{
			name:      "Test 1: normal log entry",
			logFormat: baseformat,
			inputLogEntry: `127.0.0.1 - - [12/Apr/2024:14:50:06 +0100] "GET / HTTP/1.1" 200 615 "-" ` +
				`"PostmanRuntime/7.36.1" "-" "853" "226" "0.000" "-" "HTTP/1.1" "-""-" "-" "-"`,
			expOutput: map[string]string{
				"BASE10NUM":       "853",
				"body_bytes_sent": "615",
				"bytes_sent":      "853",
				"DEFAULT": `127.0.0.1 - - [12/Apr/2024:14:50:06 +0100] "GET / HTTP/1.1" 200 615 ` +
					`"-" "PostmanRuntime/7.36.1" "-" "853" "226" "0.000" "-" "HTTP/1.1" "-""-" "-" "-"`,
				"gzip_ratio":               "-",
				"HOSTNAME":                 "",
				"http_referer":             "-",
				"http_user_agent":          "PostmanRuntime/7.36.1",
				"http_x_forwarded_for":     "-",
				"IP":                       "127.0.0.1",
				"IPV4":                     "127.0.0.1",
				"IPV6":                     "",
				"remote_addr":              "127.0.0.1",
				"remote_user":              "-",
				"request_length":           "226",
				"request_time":             "0.000",
				"request":                  "GET / HTTP/1.1",
				"server_protocol":          "HTTP/1.1",
				"status":                   "200",
				"time_local":               "12/Apr/2024:14:50:06 +0100",
				"upstream_connect_time":    "-",
				"upstream_header_time":     "-",
				"upstream_response_length": "-",
				"upstream_response_time":   "-",
			},
		},
		{
			name:      "Test 2: normal upstream log entry",
			logFormat: baseformat,
			inputLogEntry: `127.0.0.1 - - [11/Apr/2024:13:39:25 +0100] "GET /frontend1 HTTP/1.0" 200 28 "-" ` +
				`"PostmanRuntime/7.36.1" "-" "185" "222" "0.000" "-" "HTTP/1.0" "-""-" "-" "-"`,
			expOutput: map[string]string{
				"BASE10NUM": "185",
				"DEFAULT": `127.0.0.1 - - [11/Apr/2024:13:39:25 +0100] "GET /frontend1 HTTP/1.0" 200 ` +
					`28 "-" "PostmanRuntime/7.36.1" "-" "185" "222" "0.000" "-" "HTTP/1.0" "-""-" "-" "-"`,
				"HOSTNAME":                 "",
				"IP":                       "127.0.0.1",
				"IPV4":                     "127.0.0.1",
				"IPV6":                     "",
				"body_bytes_sent":          "28",
				"bytes_sent":               "185",
				"gzip_ratio":               "-",
				"http_referer":             "-",
				"http_user_agent":          "PostmanRuntime/7.36.1",
				"http_x_forwarded_for":     "-",
				"remote_addr":              "127.0.0.1",
				"remote_user":              "-",
				"request":                  "GET /frontend1 HTTP/1.0",
				"request_length":           "222",
				"request_time":             "0.000",
				"server_protocol":          "HTTP/1.0",
				"status":                   "200",
				"time_local":               "11/Apr/2024:13:39:25 +0100",
				"upstream_connect_time":    "-",
				"upstream_header_time":     "-",
				"upstream_response_length": "-",
				"upstream_response_time":   "-",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			grok, err := NewCompiledGrok(test.logFormat, logger)
			require.NoError(tt, err)
			// Our code only uses ParseString(), so we will ignore other methods in the Grok API.
			actualOutput := grok.ParseString(test.inputLogEntry)
			removeTimeEntries(actualOutput)

			assert.Equal(tt, test.expOutput, actualOutput)
		})
	}
}

func newLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logCfg := zap.NewDevelopmentConfig()
	logCfg.OutputPaths = []string{"stdout"}
	logCfg.ErrorOutputPaths = []string{"stderr"}
	logger, err := logCfg.Build()
	require.NoError(t, err)

	return logger
}

// Omit time entries, as they will vary depending on when the test will be run.
func removeTimeEntries(grokOutput map[string]string) {
	dateKeys := []string{
		"HOUR",
		"INT",
		"MINUTE",
		"MONTH",
		"MONTHDAY",
		"SECOND",
		"TIME",
		"YEAR",
	}

	for _, key := range dateKeys {
		delete(grokOutput, key)
	}
}
