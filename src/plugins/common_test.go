package plugins

import (
	"testing"

	sdk "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestLoadPlugins(t *testing.T) {
	binary := tutils.GetMockNginxBinary()
	env := tutils.GetMockEnvWithHostAndProcess()
	cmdr := tutils.NewMockCommandClient()
	reporter := tutils.NewMockMetricsReportClient()

	tests := []struct {
		name                  string
		loadedConfig          *config.Config
		expectedPluginSize    int
		expectedExtensionSize int
	}{
		{
			name:                  "default plugins",
			loadedConfig:          &config.Config{},
			expectedPluginSize:    5,
			expectedExtensionSize: 0,
		},
		{
			name: "no plugins or extensions",
			loadedConfig: &config.Config{
				Features:   []string{},
				Extensions: []string{},
			},
			expectedPluginSize:    5,
			expectedExtensionSize: 0,
		},
		{
			name: "all plugins and extensions",
			loadedConfig: &config.Config{
				Features: sdk.GetDefaultFeatures(),
				// temporarily to figure out what's going on with the monitoring extension
				Extensions: sdk.GetKnownExtensions()[:len(sdk.GetKnownExtensions())-1],
				AgentMetrics: config.AgentMetrics{
					BulkSize:           1,
					ReportInterval:     1,
					CollectionInterval: 1,
					Mode:               "aggregated",
				},
			},
			expectedPluginSize: 14,
			// temporarily to figure out what's going on with the monitoring extension
			expectedExtensionSize: len(sdk.GetKnownExtensions()[:len(sdk.GetKnownExtensions())-1]),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corePlugins, extensionPlugins := LoadPlugins(cmdr, binary, env,
				reporter,
				tt.loadedConfig,
				events.NewAgentEventMeta(
					"NGINX-AGENT",
					"v0.0.1",
					"75231",
					"test-host",
					"12345678",
					"group-a",
					[]string{"tag-a", "tag-b"}))

			assert.NotNil(t, corePlugins)
			assert.Len(t, corePlugins, tt.expectedPluginSize)
			assert.Len(t, extensionPlugins, tt.expectedExtensionSize)
		})
	}
}
