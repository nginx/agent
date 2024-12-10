// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/stub"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUUIDv7Regex(t *testing.T) {
	// Define the UUIDv7 regex
	uuidv7Regex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	// Define test cases
	tests := []struct {
		name        string
		input       string
		expectMatch bool
	}{
		{
			name:        "Valid UUIDv7 with variant 8",
			input:       "01876395-3f9d-7c91-89a3-4f57e53a1a4b",
			expectMatch: true,
		},
		{
			name:        "Valid UUIDv7 with variant a",
			input:       "01876395-3f9d-7c91-9f00-4f57e53a1a4b",
			expectMatch: true,
		},
		{
			name:        "Invalid UUIDv7 - wrong version",
			input:       "01876395-3f9d-6c91-89a3-4f57e53a1a4b",
			expectMatch: false,
		},
		{
			name:        "Invalid UUIDv7 - wrong variant",
			input:       "01876395-3f9d-7c91-7a00-4f57e53a1a4b",
			expectMatch: false,
		},
		{
			name:        "Invalid UUIDv7 - extra characters",
			input:       "01876395-3f9d-7c91-89a3-4f57e53a1a4b123",
			expectMatch: false,
		},
		{
			name:        "Invalid UUIDv7 - missing characters",
			input:       "01876395-3f9d-7c91-89a3-4f57e53a",
			expectMatch: false,
		},
	}

	// Iterate over test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := uuidv7Regex.MatchString(tt.input)
			assert.Equal(t, tt.expectMatch, match, "Regex match result did not match expectation")
		})
	}
}

func TestGenerateMessageID(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func() (uuid.UUID, error)
		expected    string
		expectError bool
	}{
		{
			name: "Valid UUID generation",
			mockFunc: func() (uuid.UUID, error) {
				return uuid.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, nil
			},
			expected:    "01020304-0506-0708-090a-0b0c0d0e0f10",
			expectError: false,
		},
		{
			name: "Fallback UUID generation due to error",
			mockFunc: func() (uuid.UUID, error) {
				return uuid.Nil, errors.New("mock error")
			},
			expected:    "", // Fallback UUIDs don't follow a fixed prefix but should not be empty
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultUUIDGenerator = tt.mockFunc

			if tt.expectError {
				logBuf := &bytes.Buffer{}
				stub.StubLoggerWith(logBuf)

				got := GenerateMessageID()
				assert.NotEmpty(t, got)

				// Inspect logs
				helpers.ValidateLog(t, "Issue generating uuidv7, using sha256 and timestamp instead", logBuf)

				logBuf.Reset()
			} else {
				got := GenerateMessageID()

				assert.Equal(t, tt.expected, got, "Expected UUID string to match")
			}

			// reset
			defaultUUIDGenerator = uuid.NewUUID
		})
	}
	defaultUUIDGenerator = func() (uuid.UUID, error) {
		return uuid.UUID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, nil
	}

	got := GenerateMessageID()
	assert.Equal(t, "01020304-0506-0708-090a-0b0c0d0e0f10", got, "Expected correct UUID string")
}
