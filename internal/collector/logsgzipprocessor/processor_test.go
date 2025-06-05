// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package logsgzipprocessor

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap"
)

var dummyInputStr = "hello world"

func TestGzipProcessor(t *testing.T) {
	testCases := []struct {
		input any
		name  string
	}{
		{
			name:  "Test 1: string content",
			input: dummyInputStr,
		},
		{
			name:  "Test 2: byte content",
			input: []byte("binary data"),
		},
		{
			name:  "Test 3: integer content",
			input: 12345,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			settings := processortest.NewNopSettings(processortest.NopType)
			settings.Logger = zap.NewNop()
			// Setup: create a log record with the test case content
			logs := plog.NewLogs()
			logRecord := logs.ResourceLogs().AppendEmpty().
				ScopeLogs().AppendEmpty().
				LogRecords().AppendEmpty()
			var expectNoOutput bool
			switch v := tc.input.(type) {
			case string:
				logRecord.Body().SetStr(v)
			case []byte:
				logRecord.Body().SetEmptyBytes().FromRaw(v)
			case int:
				logRecord.Body().SetInt(int64(v))
				expectNoOutput = true
			}

			next := &consumertest.LogsSink{}
			processor := newLogsGzipProcessor(next, settings)
			require.NoError(t, processor.Start(ctx, nil))

			capability := processor.Capabilities()
			assert.True(t, capability.MutatesData, "logs mutation should be a capability")

			// process logs
			err := processor.ConsumeLogs(ctx, logs)
			require.NoError(t, err, "processor failed")

			// output should be gzipped
			if expectNoOutput {
				assert.Equal(t, 0, next.LogRecordCount(), "no logs should be produced")
				return
			}
			got := next.AllLogs()[0]
			record := got.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			gzipped := record.Body().Bytes().AsRaw()

			// Decompress and check content
			verifyGzippedContent(t, gzipped, tc.input)
			require.NoError(t, processor.Shutdown(ctx))
		})
	}
}

type mockGzipWriter struct {
	WriteFunc func(p []byte) (int, error)
	CloseFunc func() error
	ResetFunc func(w io.Writer)
}

func (m *mockGzipWriter) Write(p []byte) (int, error) {
	if m.WriteFunc != nil {
		return m.WriteFunc(p)
	}

	return len(p), nil
}

func (m *mockGzipWriter) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	return nil
}

func (m *mockGzipWriter) Reset(w io.Writer) {
	if m.ResetFunc != nil {
		m.ResetFunc(w)
	}
}

func TestGzipProcessorFailure(t *testing.T) {
	testCases := []struct {
		name             string
		isGzipWriteError bool
		isGzipCloseError bool
	}{
		{
			name:             "Test 1: gzip write failure",
			isGzipWriteError: true,
		},
		{
			name:             "Test 2: gzip writer close failure",
			isGzipCloseError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			settings := processortest.NewNopSettings(processortest.NopType)
			settings.Logger = zap.NewNop()
			// Setup: create a log record with the test case content
			logs := plog.NewLogs()
			logRecord := logs.ResourceLogs().AppendEmpty().
				ScopeLogs().AppendEmpty().
				LogRecords().AppendEmpty()
			logRecord.Body().SetStr(dummyInputStr)

			next := &consumertest.LogsSink{}

			mockWriter := customMockWriter(tc.isGzipWriteError, tc.isGzipCloseError)
			// explicitly set writer that fails
			processor := &logsGzipProcessor{
				nextConsumer: next,
				pool: &sync.Pool{
					New: func() any {
						return mockWriter
					},
				},
				settings: settings,
			}
			require.NoError(t, processor.Start(ctx, nil))

			err := processor.ConsumeLogs(ctx, logs)
			require.Error(t, err, "processor should return error when gzip writer fails")
			require.Contains(t, err.Error(), "failed processing log records",
				"processor should return relevant error")

			require.NoError(t, processor.Shutdown(ctx))
		})
	}
}

// nolint: revive
func customMockWriter(isGzipWriteErr, isGzipCloseErr bool) *mockGzipWriter {
	return &mockGzipWriter{
		WriteFunc: func(p []byte) (int, error) {
			if isGzipWriteErr {
				return 0, errors.New("mock write error")
			}

			return 0, nil
		},
		CloseFunc: func() error {
			if isGzipCloseErr {
				return errors.New("mock close error")
			}

			return nil
		},
	}
}

func verifyGzippedContent(t *testing.T, gzipped []byte, input any) {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(gzipped))
	require.NoError(t, err, "failed to read gzipped content")
	defer gr.Close()
	plain, err := io.ReadAll(gr)
	require.NoError(t, err, "failed to read decompress content")

	// check if plain text is as expected
	switch v := input.(type) {
	case string:
		assert.Equal(t, v, string(plain))
	case []byte:
		assert.Equal(t, v, plain)
	}
}
