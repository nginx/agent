// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ValidateLog checks if the expected log message is present in the provided log buffer.
// If the expected log message is not found, it reports an error to the testing framework.
//
// Parameters:
//   - t (*testing.T): The testing object used to report errors.
//   - expectedLog (string): The expected log message to validate against the log buffer.
//     If empty, no validation is performed.
//   - logBuf (*bytes.Buffer): The log buffer that contains the actual log messages.
//
// Behavior:
// - If the expected log message is not an empty string:
//   - The function checks whether the log buffer contains the expected log message.
//   - If the log message is missing, an error is reported using t.Errorf.
//
// Usage:
// - Use this function within test cases to validate that a specific log message was produced.
//
// Example:
//
//	var logBuffer bytes.Buffer
//	logBuffer.WriteString("App started successfully")
//	ValidateLog(t, "App started successfully", &logBuffer)
func ValidateLog(t *testing.T, expectedLog string, logBuf *bytes.Buffer) {
	t.Helper()

	if expectedLog != "" {
		require.NotEmpty(t, logBuf)

		if !strings.Contains(logBuf.String(), expectedLog) {
			t.Errorf("Expected log to contain %q, but got %q", expectedLog, logBuf.String())
		}
	}
}
