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
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFeatures_Process(t *testing.T) {
	testCases := []struct {
		testName   string
		featureKey string
		pluginName string
		numPlugins int
	}{
		{
			testName:   "API",
			featureKey: agent_config.FeatureAgentAPI,
			pluginName: agent_config.FeatureAgentAPI,
			numPlugins: 2,
		},
		{
			testName:   "Metrics",
			featureKey: agent_config.FeatureMetrics,
			pluginName: agent_config.FeatureMetrics,
			numPlugins: 4,
		},
		{
			testName:   "Metrics collection",
			featureKey: agent_config.FeatureMetricsCollection,
			pluginName: agent_config.FeatureMetrics,
			numPlugins: 2,
		},
		{
			testName:   "File Watcher",
			featureKey: agent_config.FeatureFileWatcher,
			pluginName: agent_config.FeatureFileWatcher,
			numPlugins: 3,
		},
	}

	processID := "12345"

	processes := []*core.Process{
		{
			Name:     processID,
			IsMaster: true,
		},
	}

	detailsMap := map[string]*proto.NginxDetails{
		processID: {
			ProcessPath: "/path/to/nginx",
			NginxId:     processID,
			Plus: &proto.NginxPlusMetaData{
				Enabled: true,
			},
		},
	}

	_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer cleanupFunc()

	ctx, cancelCTX := context.WithCancel(context.Background())
	binary := tutils.NewMockNginxBinary()
	env := tutils.NewMockEnvironment()

	env.Mock.On("IsContainer").Return(true)
	env.On("NewHostInfo", "agentVersion", &[]string{"locally-tagged", "tagged-locally"}).Return(&proto.HostInfo{})

	binary.On("GetNginxDetailsFromProcess", &core.Process{Name: "12345", IsMaster: true}).Return(detailsMap[processID])
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(detailsMap)
	binary.On("UpdateNginxDetailsFromProcesses", mock.Anything).Return()

	cmdr := tutils.NewMockCommandClient()

	configuration, _ := config.GetConfig("1234")

	pluginUnderTest := NewFeatures(cmdr, configuration, env, binary, "agentVersion", processes, events.NewAgentEventMeta(
		config.MODULE,
		"v0.0.1",
		"75231",
		"test-host",
		"12345678",
		"group-a",
		[]string{"tag-a", "tag-b"}),
	)

	for _, tc := range testCases {
		messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

		assert.Len(t, messagePipe.GetPlugins(), 1)
		assert.Equal(t, agent_config.FeaturesPlugin, messagePipe.GetPlugins()[0].Info().Name())

		messagePipe.Process(core.NewMessage(core.EnableFeature, []string{tc.featureKey}))
		messagePipe.Run()
		time.Sleep(250 * time.Millisecond)

		processedMessages := messagePipe.GetProcessedMessages()
		assert.Equal(t, tc.numPlugins, len(messagePipe.GetPlugins()))
		assert.GreaterOrEqual(t, len(processedMessages), 1)
		assert.Equal(t, core.EnableFeature, processedMessages[0].Topic())
		assert.Equal(t, tc.pluginName, messagePipe.GetPlugins()[1].Info().Name())
	}

	cancelCTX()
	pluginUnderTest.Close()
}
