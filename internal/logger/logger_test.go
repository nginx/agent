// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package logger

import (
	helpers "github.com/nginx/agent/v3/test"
	"io"
	"log/slog"
	"os"
	"testing"

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
			name:     "Debug level",
			input:    "Debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "Info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "Warn level",
			input:    "Warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "Error level",
			input:    "ERROR",
			expected: slog.LevelError,
		},
		{
			name:     "Unknown level",
			input:    "Unknown",
			expected: slog.LevelInfo,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := getLogLevel(test.input)
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
		name     string
		input    string
		expected io.Writer
	}{
		{
			name:     "No log file",
			input:    "",
			expected: os.Stderr,
		},
		{
			name:     "Log file does not exist",
			input:    "/unknown/file.log",
			expected: os.Stderr,
		},
		{
			name:     "Log file exists",
			input:    file.Name(),
			expected: &os.File{},
		},
		{
			name:     "Log directory",
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
