// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:staticcheck
func TestRemoveASCIIControlSignals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No control characters",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name: "With control characters",
			input: "Hello, World!", expected: "Hello, World!",
		},
		{
			name: "Only control characters",
			input: "", expected: "",
		},
		{
			name:     "Mixed printable and control characters",
			input:    "Hello\nWorld\t!",
			expected: "HelloWorld!",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name: "Agent version example",
			input: "nginx-agent version v3.0.0-4a64a94", expected: "nginx-agent version v3.0.0-4a64a94",
		},
		{
			name:     "Agent version example alpine",
			input:    "#nginx-agent version v3.0.0-f94d93a",
			expected: "nginx-agent version v3.0.0-f94d93a",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RemoveASCIIControlSignals(t, test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
