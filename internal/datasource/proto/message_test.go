// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
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
