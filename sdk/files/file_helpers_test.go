/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package files

import (
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

func TestGetFileMode(t *testing.T) {
	fileModeTests := []struct {
		input    string
		expected os.FileMode
	}{
		{
			input:    "0644",
			expected: os.FileMode(0o644),
		},
		{
			input:    "0",
			expected: os.FileMode(0),
		},
		{
			input:    "0777",
			expected: os.FileMode(0o777),
		},
		{
			input:    "0234",
			expected: os.FileMode(0o234),
		},
		{
			input:    "invalid",
			expected: os.FileMode(0o644),
		},
	}

	for _, test := range fileModeTests {
		result := GetFileMode(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestGetPermissions(t *testing.T) {
	fileModeTests := []struct {
		input    os.FileMode
		expected string
	}{
		{
			input:    os.FileMode(0o644),
			expected: "0644",
		},
		{
			input:    os.FileMode(0),
			expected: "0",
		},
		{
			input:    os.FileMode(0o777),
			expected: "0777",
		},
		{
			input:    os.FileMode(0o234),
			expected: "0234",
		},
	}

	for _, test := range fileModeTests {
		result := GetPermissions(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestTimeConvert(t *testing.T) {
	fileModeTests := []struct {
		input    time.Time
		expected *types.Timestamp
	}{
		{
			input: time.Date(2022, 0o1, 23, 12, 0, 20, 100, &time.Location{}),
			expected: &types.Timestamp{
				Seconds: 1642939220,
				Nanos:   100,
			},
		},
		{
			input:    time.Date(-2022, 0o1, 23, 12, 0, 20, 100, &time.Location{}),
			expected: types.TimestampNow(),
		},
	}

	for _, test := range fileModeTests {
		result := TimeConvert(test.input)
		assert.Equal(t, test.expected.Seconds, result.Seconds)
	}
}
