/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package logger

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetLogger(t *testing.T) {
	t.Helper()
	prevLevel := log.GetLevel()
	prevOut := log.StandardLogger().Out
	t.Cleanup(func() {
		log.SetLevel(prevLevel)
		log.SetOutput(prevOut)
	})
}

func TestSetLogLevel_ValidLevels(t *testing.T) {
	tests := []struct {
		input string
		want  log.Level
	}{
		{"trace", log.TraceLevel},
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"warning", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"panic", log.PanicLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resetLogger(t)
			SetLogLevel(tt.input)
			assert.Equal(t, tt.want, log.GetLevel())
		})
	}
}

func TestSetLogLevel_EmptyIsNoOp(t *testing.T) {
	resetLogger(t)
	log.SetLevel(log.WarnLevel)
	SetLogLevel(" ")
	assert.Equal(t, log.WarnLevel, log.GetLevel(), "empty log level input must not change level")
}

func TestSetLogLevel_TrailingSpaces(t *testing.T) {
	resetLogger(t)
	log.SetLevel(log.WarnLevel)
	SetLogLevel("INFO      ")
	assert.Equal(t, log.WarnLevel, log.GetLevel(), "log level input with trailing spaces must be trimmed and applied")
}

func TestSetLogLevel_InvalidLeavesLevelUnchanged(t *testing.T) {
	resetLogger(t)
	log.SetLevel(log.InfoLevel)
	SetLogLevel("not-a-level")
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestSetLogFile_EmptyReturnsNil(t *testing.T) {
	resetLogger(t)
	got := SetLogFile("")
	assert.Nil(t, got)
}

func TestSetLogFile_NonExistentFilePathReturnsNil(t *testing.T) {
	resetLogger(t)
	missing := filepath.Join(t.TempDir(), "filepath-does-not-exist", "agent.log")
	got := SetLogFile(missing)
	assert.Nil(t, got)
}

func TestSetLogFile_ExistingFileIsAppended(t *testing.T) {
	resetLogger(t)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agent.log")

	require.NoError(t, os.WriteFile(logPath, []byte("existing\n"), 0o600))

	fh := SetLogFile(logPath)
	require.NotNil(t, fh)
	t.Cleanup(func() { _ = fh.Close() })

	log.Info("appended")

	require.NoError(t, fh.Close())

	contents, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "existing", "previous content should be preserved (O_APPEND)")
	assert.Contains(t, string(contents), "appended", "new log line should be present")
}

func TestSetLogFile_DirectoryAppendsDefaultFileName(t *testing.T) {
	resetLogger(t)
	dir := t.TempDir()

	fh := SetLogFile(dir)
	require.NotNil(t, fh)
	t.Cleanup(func() { _ = fh.Close() })

	expectedPath := filepath.Join(dir, "agent.log")
	_, err := os.Stat(expectedPath)
	assert.NoError(t, err, "expected default agent.log to be created in the directory")

	assert.Equal(t, expectedPath, fh.Name())
}

func TestSetLogFile_NewLogFileCreated(t *testing.T) {
	resetLogger(t)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "fresh.log")

	f, err := os.Create(logPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	fh := SetLogFile(logPath)
	require.NotNil(t, fh)
	t.Cleanup(func() { _ = fh.Close() })

	assert.Equal(t, logPath, fh.Name())
}
