// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypes_IsDirectoryAllowed(t *testing.T) {
	config := agentConfig()

	tests := []struct {
		name        string
		fileDir     string
		allowedDirs []string
		allowed     bool
	}{
		{
			name:    "Test 1: directory allowed",
			allowed: true,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx",
				"/var/log/nginx/",
			},
			fileDir: "/etc/nginx/nginx.conf",
		},
		{
			name:    "Test 2: directory not allowed",
			allowed: false,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx/",
				"/var/log/nginx/",
			},
			fileDir: "/etc/nginx-test/nginx-agent.conf",
		},
		{
			name:    "Test 3: directory allowed",
			allowed: true,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx/",
				"/var/log/nginx/",
			},
			fileDir: "/etc/nginx/conf.d/nginx-agent.conf",
		},
		{
			name:    "Test 4: directory not allowed",
			allowed: false,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx",
				"/var/log/nginx",
			},
			fileDir: "~/test.conf",
		},
		{
			name:    "Test 5: directory not allowed",
			allowed: false,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx/",
				"/var/log/nginx/",
			},
			fileDir: "//test.conf",
		},
		{
			name:    "Test 6: directory allowed",
			allowed: true,
			allowedDirs: []string{
				AgentDirName,
				"/etc/nginx/",
				"/var/log/nginx/",
				"/",
			},
			fileDir: "/test.conf",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config.AllowedDirectories = test.allowedDirs
			result := config.IsDirectoryAllowed(test.fileDir)
			assert.Equal(t, test.allowed, result)
		})
	}
}
