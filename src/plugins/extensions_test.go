/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"testing"
	"time"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestExtensions_Process(t *testing.T) {
	testCases := []struct {
		testName      string
		extensionKey  string
		extensionName string
	}{
		{
			testName:      "Advanced Metrics",
			extensionKey:  agent_config.AdvancedMetricsExtensionPlugin,
			extensionName: agent_config.AdvancedMetricsExtensionPlugin,
		},
		{
			testName:      "Nginx App Protect",
			extensionKey:  agent_config.NginxAppProtectExtensionPlugin,
			extensionName: agent_config.NginxAppProtectExtensionPlugin,
		},
	}

	// Create an agent config and initialize Viper config properties
	// based off of it, clean up when done.
	// TODO: The test agent config is going to be geting modified.
	// Need to either not run parallel or properly lock the code.
	_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer cleanupFunc()

	ctx, cancelCTX := context.WithCancel(context.Background())
	env := tutils.NewMockEnvironment()

	configuration, _ := config.GetConfig("1234")

	env.On("NewHostInfo", "agentVersion", &[]string{"locally-tagged", "tagged-locally"}).Return(&proto.HostInfo{})

	pluginUnderTest := NewExtensions(configuration, env)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			// Assert that only the extensions plugin is registered
			assert.Equal(t, 1, len(messagePipe.GetPlugins()))
			assert.Equal(t, "Extensions Plugin", messagePipe.GetPlugins()[0].Info().Name())

			messagePipe.Process(core.NewMessage(core.EnableExtension, tc.extensionKey))
			messagePipe.Run()
			time.Sleep(250 * time.Millisecond)

			processedMessages := messagePipe.GetProcessedMessages()
			assert.GreaterOrEqual(t, len(processedMessages), 1)
			assert.Equal(t, core.EnableExtension, processedMessages[0].Topic())

			assert.Equal(t, 1, len(messagePipe.GetPlugins()))
			assert.Equal(t, 1, len(messagePipe.GetExtensionPlugins()))
			assert.Equal(t, "Extensions Plugin", messagePipe.GetPlugins()[0].Info().Name())
			assert.Equal(t, tc.extensionName, messagePipe.GetExtensionPlugins()[0].Info().Name())
		})
	}

	cancelCTX()
	pluginUnderTest.Close()
}
