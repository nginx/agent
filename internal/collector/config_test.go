// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestOTelCollectorSettings(t *testing.T) {
	cfg := types.GetAgentConfig()

	settings := OTelCollectorSettings(cfg)

	assert.NotNil(t, settings.Factories, "Factories should not be nil")
	assert.Equal(t, "otel-nginx-agent", settings.BuildInfo.Command, "BuildInfo command should match")
	assert.False(t, settings.DisableGracefulShutdown, "DisableGracefulShutdown should be false")
	assert.NotNil(t, settings.ConfigProviderSettings, "ConfigProviderSettings should not be nil")
	assert.Nil(t, settings.LoggingOptions, "LoggingOptions should be nil")
	assert.False(t, settings.SkipSettingGRPCLogger, "SkipSettingGRPCLogger should be false")
}

func TestConfigProviderSettings(t *testing.T) {
	cfg := types.GetAgentConfig()
	settings := ConfigProviderSettings(cfg)

	assert.NotNil(t, settings.ResolverSettings, "ResolverSettings should not be nil")
	assert.Len(t, settings.ResolverSettings.ProviderFactories, 5, "There should be 5 provider factories")
	assert.Len(t, settings.ResolverSettings.ConverterFactories, 1, "There should be 1 converter factory")
	assert.NotEmpty(t, settings.ResolverSettings.URIs, "URIs should not be empty")
	assert.Equal(t, "/tmp/otel-collector-config.yaml", settings.ResolverSettings.URIs[0], "Default URI should match")
}

func TestGetConfig(t *testing.T) {
	// Test with environment variable set
	os.Setenv("OPENTELEMETRY_COLLECTOR_CONFIG_FILE", "/path/to/config.yaml")
	defer os.Unsetenv("OPENTELEMETRY_COLLECTOR_CONFIG_FILE")

	configURI := getConfig(nil)
	assert.Equal(t, "/path/to/config.yaml", configURI, "Config URI should match the environment variable")

	// Test without environment variable set
	os.Unsetenv("OPENTELEMETRY_COLLECTOR_CONFIG_FILE")

	configURI = getConfig(nil)
	assert.Equal(t, "/tmp/otel-collector-config.yaml", configURI, "Config URI should match the default value")
}
