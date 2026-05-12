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
			name: "empty map returns empty slice",
			m:    map[string]struct{}{},
			want: []string{},
		},
		{
			name: "single dir",
			m:    map[string]struct{}{"/etc/nginx": {}},
			want: []string{"/etc/nginx"},
		},
		{
			name: "multiple dirs",
			m: map[string]struct{}{
				"/etc/nginx":     {},
				"/usr/share/nms": {},
				"/var/log/nginx": {},
				"/usr/local/etc": {},
			},
			want: []string{"/etc/nginx", "/usr/share/nms", "/var/log/nginx", "/usr/local/etc"},
		},
		{
			name: "nil map",
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
		{"both set", "localhost", 1234, true},
		{"host empty", "", 1234, false},
		{"port zero", "localhost", 0, false},
		{"both zero", "", 0, false},
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
		{"present", []string{"metrics", "events"}, "metrics", true},
		{"present in middle", []string{"a", "b", "c"}, "b", true},
		{"absent", []string{"metrics"}, "events", false},
		{"empty list", []string{}, "metrics", false},
		{"nil list", nil, "metrics", false},
		{"empty query against empty list", []string{}, "", false},
		{"case sensitive", []string{"Metrics"}, "metrics", false},
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
		{"present", []string{"nginx-app-protect"}, "nginx-app-protect", true},
		{"absent", []string{"nginx-app-protect"}, "advanced-metrics", false},
		{"empty list", []string{}, "advanced-metrics", false},
		{"nil list", nil, "advanced-metrics", false},
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
		{"absolute path inside allowed dir", "/etc/nginx/nginx.conf", true},
		{"absolute path inside second allowed dir", "/var/log/nginx/access.log", true},
		{"absolute path matches dir prefix exactly", "/etc/nginx", true},
		{"absolute path outside allowed dirs", "/etc/passwd", false},
		{"multiple rooted directories in allowed dir", "/etc/nginx/conf.d/default.conf", true},
		{"relative path is rejected", "etc/nginx/nginx.conf", false},
		{"empty path", "", false},
		{"dot-relative", "./nginx.conf", false},
		{"prefix-only match (current behaviour)", "/etc/nginxfoo/x.conf", true},
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
