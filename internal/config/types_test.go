// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
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

func TestTypes_isAllowedDirWithSymlink(t *testing.T) {
	t.Run("Test 1: Symlink in allowed directory is not allowed", func(t *testing.T) {
		allowedDirs := []string{"/etc/nginx"}
		filePath := "file.conf"
		symlinkPath := "file_link"

		// Create a temp directory for the symlink
		tempDir := t.TempDir()
		defer os.RemoveAll(tempDir) // Clean up the temp directory after the test

		// Ensure the temp directory is in the allowedDirs
		allowedDirs = append(allowedDirs, tempDir)

		filePath = tempDir + "/" + filePath
		defer os.RemoveAll(filePath)
		err := os.WriteFile(filePath, []byte("test content"), 0o600)
		require.NoError(t, err)

		// Create a symlink for testing
		symlinkPath = tempDir + "/" + symlinkPath
		defer os.Remove(symlinkPath)
		err = os.Symlink(filePath, symlinkPath)
		require.NoError(t, err)

		result, err := isAllowedDir(symlinkPath, allowedDirs)
		require.Error(t, err)
		require.False(t, result, "Symlink in allowed directory should return false")
	})
}

func TestTypes_isSymlink(t *testing.T) {
	// create temp dir
	tempDir := t.TempDir()
	tempConf := tempDir + "test.conf"
	defer os.RemoveAll(tempDir)

	t.Run("Test 1: File is not a symlink", func(t *testing.T) {
		filePath := tempConf
		err := os.WriteFile(filePath, []byte("test content"), 0o600)
		require.NoError(t, err)
		require.False(t, isSymlink(filePath), "File is not a symlink")
	})
	t.Run("Test 2: File is a symlink", func(t *testing.T) {
		filePath := tempDir + "test_conf_link"
		err := os.Symlink(tempConf, filePath)
		require.NoError(t, err)
		require.True(t, isSymlink(filePath), "File is a symlink")
	})
}
