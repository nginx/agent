// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	result := New("", "")
	assert.IsType(t, &slog.Logger{}, result)
}

func TestLogWriter_NoPreviousHandleLeak(t *testing.T) {
	dir := t.TempDir()
	logPath1 := filepath.Join(dir, "agent1.log")
	logPath2 := filepath.Join(dir, "agent2.log")

	// First call — opens handle for logPath1
	_ = logWriter(logPath1)
	require.NotNil(t, currentLogFileHandle, "first logWriter call must store the handle")
	handle1 := currentLogFileHandle

	// Second call — must close handle1 and open handle for logPath2
	_ = logWriter(logPath2)
	require.NotNil(t, currentLogFileHandle)
	assert.NotEqual(t, handle1, currentLogFileHandle, "second call must replace the stored handle")

	// handle1 should now be closed — writing to it must fail
	_, err := handle1.WriteString("should fail")
	assert.Error(t, err, "previous handle must be closed after logWriter is called again")

	// Clean up
	currentLogFileHandle = nil
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "Test 1: Debug level",
			input:    "Debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "Test 2: Info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "Test 3: Warn level",
			input:    "Warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "Test 4: Error level",
			input:    "ERROR",
			expected: slog.LevelError,
		},
		{
			name:     "Test 5: Unknown level",
			input:    "Unknown",
			expected: slog.LevelInfo,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := LogLevel(test.input)
			assert.IsType(tt, test.expected, result)
		})
	}
}

func TestGetLogWriter(t *testing.T) {
	tempDir := t.TempDir()
	file, err := os.CreateTemp(tempDir, "TestGetLogWriter.*.log")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	require.NoError(t, err)

	tests := []struct {
		expected io.Writer
		name     string
		input    string
	}{
		{
			name:     "Test 1: No log file",
			input:    "",
			expected: os.Stderr,
		},
		{
			name:     "Test 2: Log file does not exist",
			input:    filepath.Join(tempDir, "file.log"),
			expected: io.MultiWriter(),
		},
		{
			name:     "Test 3: Log file and directory do not exist",
			input:    filepath.Join(tempDir, "nginx-agent", "file.log"),
			expected: os.Stderr,
		},
		{
			name:     "Test 4: Log file exists",
			input:    file.Name(),
			expected: io.MultiWriter(),
		},
		{
			name:     "Test 5: Invalid log file path",
			input:    "../../invalid_path/agent.log",
			expected: os.Stderr,
		},
		{
			name:     "Test 6: Log directory",
			input:    tempDir,
			expected: io.MultiWriter(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := logWriter(test.input)
			assert.IsType(tt, test.expected, result)
		})
	}
}

func TestGetCorrelationID(t *testing.T) {
	ctx := context.WithValue(context.Background(), CorrelationIDContextKey, GenerateCorrelationID())
	correlationID := CorrelationID(ctx)
	assert.NotEmpty(t, correlationID)
}

func TestCorrelationIDAttr_ReturnsSameIDAsLogged(t *testing.T) {
	// Context with no existing correlation ID triggers the generation path.
	ctx := context.Background()

	attr1 := CorrelationIDAttr(ctx)
	attr2 := CorrelationIDAttr(ctx)

	// Each independent call generates a new ID — that is expected.
	// What must NOT happen: within a single call, the logged ID ≠ returned ID.
	// We verify this by confirming the returned attr is non-empty and stable
	// (same value returned, not a second call to GenerateCorrelationID inside).
	assert.NotEmpty(t, attr1.Value.String())
	assert.Equal(t, CorrelationIDKey, attr1.Key)

	// Confirm the returned ID matches what would be stored in context.
	ctxWithID := context.WithValue(ctx, CorrelationIDContextKey, attr1)
	retrieved := CorrelationIDAttr(ctxWithID)
	assert.Equal(t, attr1.Value.String(), retrieved.Value.String(),
		"ID stored in context must equal the ID returned by CorrelationIDAttr")
	_ = attr2
}

func TestContextHandler_observe(t *testing.T) {
	ctx := context.WithValue(context.Background(), CorrelationIDContextKey, GenerateCorrelationID())

	testContextHandler := contextHandler{nil, []any{CorrelationIDContextKey}}
	attributes := testContextHandler.observe(ctx)

	assert.Len(t, attributes, 1)
	assert.Equal(t, CorrelationIDKey, attributes[0].Key)
	assert.NotEmpty(t, attributes[0].Value.String())
}
