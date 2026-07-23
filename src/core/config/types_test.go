/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_AllowedDirectories(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]struct{}
		want []string
	}{
		{
			name: "Test 1: Empty map returns empty slice",
			m:    map[string]struct{}{},
			want: []string{},
		},
		{
			name: "Test 2: Single dir",

			m:    map[string]struct{}{"/etc/nginx": {}},
			want: []string{"/etc/nginx"},
		},
		{
			name: "Test 3: Multiple dirs",
			m: map[string]struct{}{
				"/etc/nginx":     {},
				"/usr/share/nms": {},
				"/var/log/nginx": {},
				"/usr/local/etc": {},
			},
			want: []string{"/etc/nginx", "/usr/share/nms", "/var/log/nginx", "/usr/local/etc"},
		},
		{
			name: "Test 4: nil map",
			m:    nil,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{AllowedDirectoriesMap: tt.m}
			got := c.AllowedDirectories()
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestConfig_IsGrpcServerConfigured(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		port   int
		expect bool
	}{
		{
			name:   "both set",
			host:   "localhost",
			port:   1234,
			expect: true,
		},
		{
			name:   "host empty",
			host:   "",
			port:   1234,
			expect: false,
		},
		{
			name:   "port zero",
			host:   "localhost",
			port:   0,
			expect: false,
		},
		{
			name:   "both zero",
			host:   "",
			port:   0,
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Server: Server{Host: tt.host, GrpcPort: tt.port}}
			assert.Equal(t, tt.expect, c.IsGrpcServerConfigured())
		})
	}
}

func TestConfig_IsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		query    string
		expect   bool
	}{
		{
			name:     "present",
			features: []string{"metrics", "events"},
			query:    "metrics",
			expect:   true,
		},
		{
			name:     "present in middle",
			features: []string{"a", "b", "c"},
			query:    "b",
			expect:   true,
		},
		{
			name:     "absent",
			features: []string{"metrics"},
			query:    "events",
			expect:   false,
		},
		{
			name:     "empty list",
			features: []string{},
			query:    "metrics",
			expect:   false,
		},
		{
			name:     "nil list",
			features: nil,
			query:    "metrics",
			expect:   false,
		},
		{
			name:     "empty query against empty list",
			features: []string{},
			query:    "",
			expect:   false,
		},
		{
			name:     "case sensitive",
			features: []string{"Metrics"},
			query:    "metrics",
			expect:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Features: tt.features}
			assert.Equal(t, tt.expect, c.IsFeatureEnabled(tt.query))
		})
	}
}

func TestConfig_IsExtensionEnabled(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		query      string
		expect     bool
	}{
		{
			name:       "present",
			extensions: []string{"nginx-app-protect"},
			query:      "nginx-app-protect",
			expect:     true,
		},
		{
			name:       "absent",
			extensions: []string{"nginx-app-protect"},
			query:      "advanced-metrics",
			expect:     false,
		},
		{
			name:       "empty list",
			extensions: []string{},
			query:      "advanced-metrics",
			expect:     false,
		},
		{
			name:       "nil list",
			extensions: nil,
			query:      "advanced-metrics",
			expect:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Extensions: tt.extensions}
			assert.Equal(t, tt.expect, c.IsExtensionEnabled(tt.query))
		})
	}
}

func TestConfig_GetServerBackoffSettings_MapsAllFields(t *testing.T) {
	c := &Config{
		Server: Server{
			Backoff: Backoff{
				InitialInterval:     1 * time.Second,
				RandomizationFactor: 0.25,
				Multiplier:          2.0,
				MaxInterval:         30 * time.Second,
				MaxElapsedTime:      5 * time.Minute,
			},
		},
	}

	got := c.GetServerBackoffSettings()

	assert.Equal(t, 1*time.Second, got.InitialInterval)
	assert.Equal(t, 30*time.Second, got.MaxInterval)
	assert.Equal(t, 5*time.Minute, got.MaxElapsedTime)
	assert.InDelta(t, 2.0, got.Multiplier, 0)
	assert.InDelta(t, 0.25, got.Jitter, 0)
}

func TestConfig_GetMetricsBackoffSettings_MapsAllFields(t *testing.T) {
	c := &Config{
		AgentMetrics: AgentMetrics{
			Backoff: Backoff{
				InitialInterval:     500 * time.Millisecond,
				RandomizationFactor: 0.1,
				Multiplier:          1.5,
				MaxInterval:         10 * time.Second,
				MaxElapsedTime:      1 * time.Minute,
			},
		},
	}

	got := c.GetMetricsBackoffSettings()

	assert.Equal(t, 500*time.Millisecond, got.InitialInterval)
	assert.Equal(t, 10*time.Second, got.MaxInterval)
	assert.Equal(t, 1*time.Minute, got.MaxElapsedTime)
	assert.InDelta(t, 1.5, got.Multiplier, 0)
	assert.InDelta(t, 0.1, got.Jitter, 0)
}

func TestConfig_GetServerBackoffSettings_ZeroValues(t *testing.T) {
	c := &Config{}
	got := c.GetServerBackoffSettings()

	assert.Zero(t, got.InitialInterval)
	assert.Zero(t, got.MaxInterval)
	assert.Zero(t, got.MaxElapsedTime)
	assert.InDelta(t, 0.0, got.Multiplier, 0)
	assert.InDelta(t, 0.0, got.Jitter, 0)
}

func TestConfig_IsFileAllowed(t *testing.T) {
	allowed := map[string]struct{}{
		"/etc/nginx":     {},
		"/var/log/nginx": {},
	}

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{
			name:   "absolute path inside allowed dir",
			path:   "/etc/nginx/nginx.conf",
			expect: true,
		},
		{
			name:   "absolute path inside second allowed dir",
			path:   "/var/log/nginx/access.log",
			expect: true,
		},
		{
			name:   "absolute path matches dir prefix exactly",
			path:   "/etc/nginx",
			expect: true,
		},
		{
			name:   "absolute path outside allowed dirs",
			path:   "/etc/passwd",
			expect: false,
		},
		{
			name:   "multiple rooted directories in allowed dir",
			path:   "/etc/nginx/conf.d/default.conf",
			expect: true,
		},
		{
			name:   "relative path is rejected",
			path:   "etc/nginx/nginx.conf",
			expect: false,
		},
		{
			name:   "empty path",
			path:   "",
			expect: false,
		},
		{
			name:   "dot-relative",
			path:   "./nginx.conf",
			expect: false,
		},
		{
			name:   "dot-relative",
			path:   "../etc/nginx.conf",
			expect: false,
		},
		{
			name:   "prefix-only match",
			path:   "/etc/nginxfoo/x.conf",
			expect: false,
		},
		{
			name:   "path traversal via allowed dir is rejected",
			path:   "/etc/nginx/../../etc/passwd",
			expect: false,
		},
		{
			name:   "absolute path in unrelated dir is rejected",
			path:   "/tmp/nginx/nginx.conf",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{AllowedDirectoriesMap: allowed}
			assert.Equal(t, tt.expect, c.IsFileAllowed(tt.path))
		})
	}
}

func TestConfig_IsFileAllowed_EmptyAllowList(t *testing.T) {
	c := &Config{AllowedDirectoriesMap: map[string]struct{}{}}
	assert.False(t, c.IsFileAllowed("/etc/nginx/nginx.conf"))
}

func TestConfig_IsFileAllowed_NilAllowList(t *testing.T) {
	c := &Config{AllowedDirectoriesMap: nil}
	assert.False(t, c.IsFileAllowed("/etc/nginx/nginx.conf"))
}
