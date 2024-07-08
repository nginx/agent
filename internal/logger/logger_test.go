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
	"testing"

	"github.com/nginx/agent/v3/test/helpers"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	result := New(config.Log{})
	assert.IsType(t, &slog.Logger{}, result)
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
			result := GetLogLevel(test.input)
			assert.IsType(tt, test.expected, result)
		})
	}
}

func TestGetLogWriter(t *testing.T) {
	file, err := os.CreateTemp(".", "TestGetLogWriter.*.log")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	defer helpers.RemoveFileWithErrorCheck(t, "agent.log")

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
			input:    "/unknown/file.log",
			expected: os.Stderr,
		},
		{
			name:     "Test 3: Log file exists",
			input:    file.Name(),
			expected: &os.File{},
		},
		{
			name:     "Test 4: Log directory",
			input:    ".",
			expected: &os.File{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := getLogWriter(test.input)
			assert.IsType(tt, test.expected, result)
		})
	}
}

func TestGetCorrelationID(t *testing.T) {
	ctx := context.WithValue(context.Background(), CorrelationIDContextKey, GenerateCorrelationID())
	correlationID := GetCorrelationID(ctx)
	assert.NotEmpty(t, correlationID)
}

func TestContextHandler_observe(t *testing.T) {
	ctx := context.WithValue(context.Background(), CorrelationIDContextKey, GenerateCorrelationID())

	testContextHandler := contextHandler{nil, []any{CorrelationIDContextKey}}
	attributes := testContextHandler.observe(ctx)

	assert.Len(t, attributes, 1)
	assert.Equal(t, CorrelationIDKey, attributes[0].Key)
	assert.NotEmpty(t, attributes[0].Value.String())
}
