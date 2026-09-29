// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"

	// Register the stanza operators used by the NAP tcplog receiver.
	_ "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/parser/syslog"
	_ "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/transformer/add"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/types"
)

// TestCollector_NAPSyslogTimestampConversion runs NAP syslog messages through the
// timestamp conversion and syslog parser operators generated for the NAP tcplog
// receiver, and checks that the resulting log record timestamp is the correct UTC
// instant regardless of the day of month or the offset of the NAP host.
func TestCollector_NAPSyslogTimestampConversion(t *testing.T) {
	year := time.Now().Year()

	tests := []struct {
		name      string
		timestamp string
		expected  string
	}{
		{
			name:      "Test 1: single digit day, positive offset",
			timestamp: "%d-09-04T12:36:59+06:00",
			expected:  "%d-09-04T06:36:59Z",
		},
		{
			name:      "Test 2: double digit day, positive offset",
			timestamp: "%d-09-24T12:36:59+06:00",
			expected:  "%d-09-24T06:36:59Z",
		},
		{
			name:      "Test 3: first double digit day, positive offset",
			timestamp: "%d-09-10T12:36:59+06:00",
			expected:  "%d-09-10T06:36:59Z",
		},
		{
			name:      "Test 4: single digit day, negative offset",
			timestamp: "%d-03-04T12:36:59-05:00",
			expected:  "%d-03-04T17:36:59Z",
		},
		{
			name:      "Test 5: double digit day, negative offset",
			timestamp: "%d-03-31T12:36:59-05:00",
			expected:  "%d-03-31T17:36:59Z",
		},
		{
			name:      "Test 6: single digit day, UTC",
			timestamp: "%d-09-04T12:36:59Z",
			expected:  "%d-09-04T12:36:59Z",
		},
		{
			name:      "Test 7: double digit day, UTC",
			timestamp: "%d-09-24T12:36:59+00:00",
			expected:  "%d-09-24T12:36:59Z",
		},
		{
			name:      "Test 8: conversion to UTC moves to previous day",
			timestamp: "%d-10-01T03:00:00+06:00",
			expected:  "%d-09-30T21:00:00Z",
		},
		{
			name:      "Test 9: conversion to UTC moves to next day",
			timestamp: "%d-06-09T22:15:00-05:00",
			expected:  "%d-06-10T03:15:00Z",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			body := fmt.Sprintf("<130>%s ip-172-16-0-53 ASM:attack_type=\"SQL-Injection\"",
				fmt.Sprintf(test.timestamp, year))

			processed := processNAPSyslogMessage(tt, body)

			expected, err := time.Parse(time.RFC3339, fmt.Sprintf(test.expected, year))
			require.NoError(tt, err)
			assert.Equal(tt, expected.UTC(), processed.Timestamp.UTC(),
				"body sent to syslog parser: %v", processed.Body)
		})
	}
}

// processNAPSyslogMessage runs a NAP syslog message through the timestamp conversion
// and syslog parser operators generated for the NAP tcplog receiver, and returns the
// processed entry.
func processNAPSyslogMessage(t *testing.T, body string) *entry.Entry {
	t.Helper()

	operators := buildNAPTimestampOperators(t)
	output := testutil.NewFakeOutput(t)
	var next operator.Operator = output
	for i := len(operators) - 1; i >= 0; i-- {
		operators[i].SetOutputIDs([]string{next.ID()})
		require.NoError(t, operators[i].SetOutputs([]operator.Operator{next}))
		next = operators[i]
	}

	e := entry.New()
	e.Body = body
	require.NoError(t, next.Process(t.Context(), e))

	select {
	case processed := <-output.Received:
		return processed
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for processed entry")
		return nil
	}
}

// buildNAPTimestampOperators builds the timestamp conversion and syslog parser
// operators, in pipeline order, from the NAP tcplog receiver configuration generated
// by the collector.
func buildNAPTimestampOperators(t *testing.T) []operator.Operator {
	t.Helper()

	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	collector, err := NewCollector(conf)
	require.NoError(t, err)

	require.True(t, collector.updateNginxAppProtectTcplogReceivers(context.Background(),
		&model.NginxConfigContext{NAPSysLogServer: "localhost:15633"}))
	receiver := conf.Collector.Receivers.TcplogReceivers["nginx_app_protect"]
	require.NotNil(t, receiver)

	var operatorConfigs []config.Operator
	for _, op := range receiver.Operators {
		if op.Type == "syslog_parser" || (op.Type == "add" && op.Fields["field"] == "body") {
			operatorConfigs = append(operatorConfigs, op)
		}
	}
	require.Len(t, operatorConfigs, 2)

	set := componenttest.NewNopTelemetrySettings()
	operators := make([]operator.Operator, 0, len(operatorConfigs))
	for _, opConfig := range operatorConfigs {
		op, buildErr := decodeOperatorConfig(t, opConfig).Build(set)
		require.NoError(t, buildErr)
		operators = append(operators, op)
	}

	return operators
}

// decodeOperatorConfig renders an operator the same way otelcol.tmpl does and
// decodes it into a stanza operator config, so field values go through the same
// YAML unquoting as the generated collector configuration.
func decodeOperatorConfig(t *testing.T, op config.Operator) operator.Config {
	t.Helper()

	var rendered strings.Builder
	fmt.Fprintf(&rendered, "type: %s\n", op.Type)
	for key, value := range op.Fields {
		fmt.Fprintf(&rendered, "%s: %s\n", key, value)
	}

	var raw map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(rendered.String()), &raw))

	rawJSON, err := json.Marshal(raw)
	require.NoError(t, err)

	var cfg operator.Config
	require.NoError(t, json.Unmarshal(rawJSON, &cfg))

	return cfg
}
