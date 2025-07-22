// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypes_isAllowedDir(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		allowedDirs []string
		allowed     bool
	}{
		{
			name:    "Test 1: File is in allowed directory",
			allowed: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/nginx.conf",
		},
		{
			name:    "Test 2: File is in allowed directory with hyphen",
			allowed: true,
			allowedDirs: []string{
				"/etc/nginx-agent",
			},
			filePath: "/etc/nginx-agent/nginx.conf",
		},
		{
			name:    "Test 3: File exists and is in a subdirectory of allowed directory",
			allowed: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/conf.d/nginx.conf",
		},
		{
			name:    "Test 4: File exists and is outside allowed directory",
			allowed: false,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/test/nginx.conf",
		},
		{
			name:    "Test 5: File does not exist but is in allowed directory",
			allowed: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/idontexist.conf",
		},
		{
			name:    "Test 6: Test File does not exist and is outside allowed directory",
			allowed: false,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/not-nginx-test/idontexist.conf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := isAllowedDir(test.filePath, test.allowedDirs)
			require.NoError(t, err)
			require.Equal(t, test.allowed, result)
		})
	}
}
