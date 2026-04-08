// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsfilterprocessor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
)

// validNAPBody returns a pipe-separated syslog body with exactly 28 fields
// matching the secops_dashboard-log profile format.
//
//nolint:lll // long test string kept for readability
func validNAPBody() string {
	return `<130>Mar 11 18:12:45 ip-172-16-0-53 ASM:5377540117854870581|127.0.0.1|56064|10.0.0.1|80|1-localhost:1-/|nms_app_protect_default_policy|GET|/<><script>|HTTP|blocked|0|REJECTED|SECURITY_WAF_VIOLATION|5|N/A|false|200000099,200000093|sig1,sig2|N/A|{High Accuracy Signatures}|N/A|N/A|N/A|Illegal meta character in URL|<?xml version='1.0'?><BAD_MSG/>|GET /<><script> HTTP/1.1\r\nHost: localhost\r\n\r\n|US`
}

//nolint:lll // long test strings kept for readability
func TestSecurityViolationsFilterProcessor(t *testing.T) {
	testCases := []struct {
		name          string
		expectBody    string
		stringBody    string
		bodyType      pcommon.ValueType
		expectRecords int
	}{
		{
			name:          "Test 1: Pipe-separated NAP syslog message passes through untouched",
			expectRecords: 1,
			expectBody:    validNAPBody(),
			stringBody:    validNAPBody(),
			bodyType:      pcommon.ValueTypeStr,
		},
		{
			name:          "Test 2: Non-string body is dropped",
			expectRecords: 0,
			bodyType:      pcommon.ValueTypeInt,
		},
		{
			name:          "Test 3: Body with no pipes triggers gate closure",
			expectRecords: 0,
			stringBody:    "this is not csv at all",
			bodyType:      pcommon.ValueTypeStr,
		},
		{
			name:          "Test 4: Body with too few pipe-separated fields triggers gate closure",
			expectRecords: 0,
			stringBody:    "field1|field2|field3",
			bodyType:      pcommon.ValueTypeStr,
		},
		{
			name:          "Test 5: Body missing geo_location field triggers gate closure",
			expectRecords: 0,
			stringBody:    strings.TrimSuffix(validNAPBody(), "|US"),
			bodyType:      pcommon.ValueTypeStr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			settings := processortest.NewNopSettings(processortest.NopType)
			settings.Logger = zap.NewNop()

			logs := plog.NewLogs()
			lr := appendLogRecord(logs)
			setLogRecordBody(lr, tc.bodyType, tc.stringBody)

			sink := &consumertest.LogsSink{}
			p := newSecurityViolationsFilterProcessor(sink, settings)
			require.NoError(t, p.Start(ctx, nil))

			err := p.ConsumeLogs(ctx, logs)
			require.NoError(t, err)

			if tc.expectRecords == 0 {
				if sink.LogRecordCount() > 0 {
					got := sink.AllLogs()[0]
					rlOut := got.ResourceLogs().At(0)
					slOut := rlOut.ScopeLogs().At(0)
					assert.Equal(t, 0, slOut.LogRecords().Len(), "no log records should remain")
				}

				require.NoError(t, p.Shutdown(ctx))

				return
			}

			require.Equal(t, tc.expectRecords, sink.LogRecordCount())

			got := sink.AllLogs()[0]
			rlOut := got.ResourceLogs().At(0)

			// Verify resource-level schema attributes
			resAttrs := rlOut.Resource().Attributes()
			schemaName, ok := resAttrs.Get(csvSchemaNameKey)
			assert.True(t, ok, "csv.schema.name should be set")
			assert.Equal(t, csvSchemaName, schemaName.Str())

			schemaVersion, ok := resAttrs.Get(csvSchemaVersionKey)
			assert.True(t, ok, "csv.schema.version should be set")
			assert.Equal(t, csvSchemaVersion, schemaVersion.Str())

			// Verify body is passed through untouched
			lrOut := rlOut.ScopeLogs().At(0).LogRecords().At(0)
			assert.Equal(t, tc.expectBody, lrOut.Body().Str())

			// Verify no per-record attributes are set
			assert.Equal(t, 0, lrOut.Attributes().Len(),
				"no per-record attributes should be set by the filter processor")

			require.NoError(t, p.Shutdown(ctx))
		})
	}
}

func TestSecurityViolationsFilterProcessor_GateClosesOnNonCSV(t *testing.T) {
	ctx := context.Background()
	settings := processortest.NewNopSettings(processortest.NopType)
	settings.Logger = zap.NewNop()

	sink := &consumertest.LogsSink{}
	p := newSecurityViolationsFilterProcessor(sink, settings)
	require.NoError(t, p.Start(ctx, nil))

	// First message: no pipe separator → gate closes
	logs1 := newLogsWithStringBody("no pipes here")
	require.NoError(t, p.ConsumeLogs(ctx, logs1))

	// Second message: valid CSV but gate is already closed → still dropped
	logs2 := newLogsWithStringBody(validNAPBody())
	require.NoError(t, p.ConsumeLogs(ctx, logs2))

	// All forwarded batches should have zero log records
	for _, l := range sink.AllLogs() {
		for _, rl := range l.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				assert.Equal(t, 0, sl.LogRecords().Len(),
					"all records should be dropped when gate is closed")
			}
		}
	}

	require.NoError(t, p.Shutdown(ctx))
}

func TestSecurityViolationsFilterProcessor_GateOpensOnCSV(t *testing.T) {
	ctx := context.Background()
	settings := processortest.NewNopSettings(processortest.NopType)
	settings.Logger = zap.NewNop()

	sink := &consumertest.LogsSink{}
	p := newSecurityViolationsFilterProcessor(sink, settings)
	require.NoError(t, p.Start(ctx, nil))

	// First message: has enough pipe-separated fields → gate opens
	logs1 := newLogsWithStringBody(validNAPBody())
	require.NoError(t, p.ConsumeLogs(ctx, logs1))
	assert.Equal(t, 1, sink.LogRecordCount(), "first valid CSV record should pass through")

	// Second message: also passes
	logs2 := newLogsWithStringBody(validNAPBody())
	require.NoError(t, p.ConsumeLogs(ctx, logs2))
	assert.Equal(t, 2, sink.LogRecordCount(), "subsequent records should pass through")

	require.NoError(t, p.Shutdown(ctx))
}

func TestSecurityViolationsFilterProcessor_NonStringBodyClosesGate(t *testing.T) {
	ctx := context.Background()
	settings := processortest.NewNopSettings(processortest.NopType)
	settings.Logger = zap.NewNop()

	sink := &consumertest.LogsSink{}
	p := newSecurityViolationsFilterProcessor(sink, settings)
	require.NoError(t, p.Start(ctx, nil))

	// Send a non-string body first — gate should close
	logs1 := newLogsWithIntBody(42)
	require.NoError(t, p.ConsumeLogs(ctx, logs1))
	assert.Equal(t, 0, sink.LogRecordCount(), "non-string body should close gate")

	// Subsequent valid CSV is dropped because gate is closed
	logs2 := newLogsWithStringBody(validNAPBody())
	require.NoError(t, p.ConsumeLogs(ctx, logs2))
	assert.Equal(t, 0, sink.LogRecordCount(), "gate closed — all records dropped")

	require.NoError(t, p.Shutdown(ctx))
}

func TestSecurityViolationsFilterProcessor_Capabilities(t *testing.T) {
	p := newSecurityViolationsFilterProcessor(&consumertest.LogsSink{},
		processortest.NewNopSettings(processortest.NopType))
	assert.True(t, p.Capabilities().MutatesData)
}

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.Equal(t, "securityviolationsfilter", f.Type().String())
}

func newLogsWithStringBody(body string) plog.Logs {
	logs := plog.NewLogs()
	appendLogRecord(logs).Body().SetStr(body)

	return logs
}

func newLogsWithIntBody(body int64) plog.Logs {
	logs := plog.NewLogs()
	appendLogRecord(logs).Body().SetInt(body)

	return logs
}

func appendLogRecord(logs plog.Logs) plog.LogRecord {
	return logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
}

func setLogRecordBody(lr plog.LogRecord, bodyType pcommon.ValueType, stringBody string) {
	if bodyType == pcommon.ValueTypeInt {
		lr.Body().SetInt(12345)
	} else {
		lr.Body().SetStr(stringBody)
	}
}
