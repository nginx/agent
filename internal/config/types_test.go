// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypes_isAllowedDir(t *testing.T) {

	tests := []struct {
		name        string
		filePath    string
		dirExists   bool
		fileExists  bool
		allowedDirs []string
		allowed     bool
	}{
		{
			name:       "File exists and is in allowed directory",
			allowed:    true,
			dirExists:  true,
			fileExists: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/nginx.conf",
		},
		{
			name:       "File exists and is in allowed directory with hyphen",
			allowed:    true,
			dirExists:  true,
			fileExists: true,
			allowedDirs: []string{
				"/etc/nginx-agent",
			},
			filePath: "/etc/nginx-agent/nginx.conf",
		},
		{
			name:       "File exists and is in a subdirectory of allowed directory",
			allowed:    true,
			dirExists:  true,
			fileExists: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/conf.d/nginx.conf",
		},
		{
			name:       "File exists and is outside allowed directory",
			allowed:    false,
			dirExists:  true,
			fileExists: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/test/nginx.conf",
		},
		{
			name:       "File does not exist but is in allowed directory",
			allowed:    true,
			dirExists:  true,
			fileExists: false,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/etc/nginx/idontexist.conf",
		},
		{
			name:       "File does not exist and is outside allowed directory",
			allowed:    false,
			dirExists:  false,
			fileExists: true,
			allowedDirs: []string{
				"/etc/nginx",
			},
			filePath: "/not-nginx-test/idontexist.conf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.dirExists {
				// Create the temporary directory for testing
				tmpDir, err := os.MkdirTemp("", "test-allowed-dir")
				defer func() {
					err := os.RemoveAll(tmpDir)
					if err != nil {
						t.Fatalf("Failed to remove temp directory: %v", err)
					}
				}()
				if err != nil {
					t.Fatalf("Failed to create temp directory: %v", err)
				}

				// Prepend the temporary directory to the allowed directories for testing
				for i, dir := range test.allowedDirs {
					if dir != "" && dir[0] != '/' {
						test.allowedDirs[i] = tmpDir + "/" + dir
					} else {
						test.allowedDirs[i] = tmpDir + dir
					}
				}

				// Prepend the temporary directory to the fileDir for testing
				test.filePath = tmpDir + test.filePath

				// Create the parent directories
				if err := os.MkdirAll(filepath.Dir(test.filePath), 0755); err != nil {
					t.Fatalf("Failed to create directory for file: %v", err)
				}

				// Create the test file if it should exist
				if test.fileExists {
					if _, err := os.Create(test.filePath); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
			}
			result := isAllowedDir(test.filePath, test.allowedDirs)
			assert.Equal(t, test.allowed, result)
		})
	}
}
