/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/collectors"
	tutils "github.com/nginx/agent/v2/test/utils"
)

var (
	updateTags     = []string{"updated", "tags"}
	firstNginxId   = "223344"
	secondNginxId  = "334455"
	firstNginxPid  = "12345"
	secondNginxPid = "12346"
	detailsMap     = map[string]*proto.NginxDetails{
		firstNginxPid: {
			ProcessPath: "/path/to/nginx",
			NginxId:     firstNginxId,
			Plus: &proto.NginxPlusMetaData{
				Enabled: true,
			},
		},
		secondNginxPid: {
			ProcessPath: "/path/to/nginx",
			NginxId:     secondNginxId,
			Plus: &proto.NginxPlusMetaData{
				Enabled: true,
			},
		},
	}

	firstCollectorConfig = &metrics.NginxCollectorConfig{
		BinPath:            "/path/to/nginx",
		NginxId:            firstNginxId,
		CollectionInterval: 1,
		AccessLogs:         []string{},
		ErrorLogs:          []string{},
	}

	secondCollectorConfig = &metrics.NginxCollectorConfig{
		BinPath:            "/path/to/nginx",
		NginxId:            secondNginxId,
		CollectionInterval: 1,
		AccessLogs:         []string{},
		ErrorLogs:          []string{},
	}
)

func TestMetricsProcessNginxDetailProcUpdate(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsFromProcess", core.Process{Name: firstNginxPid, IsMaster: true}).Return(detailsMap[firstNginxPid])
	binary.On("GetNginxDetailsFromProcess", core.Process{Name: secondNginxPid, IsMaster: true}).Return(detailsMap[secondNginxPid])

	config := &config.Config{
		ClientID: "456789",
		Tags:     tutils.InitialConfTags,
		AgentMetrics: config.AgentMetrics{
			BulkSize:           100,
			ReportInterval:     10,
			CollectionInterval: 1,
			Mode:               "aggregated",
		},
	}

	testCases := []struct {
		testName                   string
		message                    *core.Message
		processes                  []core.Process
		expectedNumberOfCollectors int
		expectedCollectorConfigMap map[string]*metrics.NginxCollectorConfig
	}{
		{
			testName: "NginxRestart",
			message:  core.NewMessage(core.NginxDetailProcUpdate, []core.Process{}),
			processes: []core.Process{
				{
					Name:     firstNginxPid,
					IsMaster: true,
				},
			},
			expectedNumberOfCollectors: 1,
			expectedCollectorConfigMap: map[string]*metrics.NginxCollectorConfig{
				firstNginxId: firstCollectorConfig,
			},
		},
		{
			testName: "NewNginxInstanceAdded",
			message:  core.NewMessage(core.NginxDetailProcUpdate, []core.Process{}),
			processes: []core.Process{
				{
					Name:     firstNginxPid,
					IsMaster: true,
				},
				{
					Name:     secondNginxPid,
					IsMaster: true,
				},
			},
			expectedNumberOfCollectors: 2,
			expectedCollectorConfigMap: map[string]*metrics.NginxCollectorConfig{
				firstNginxId:  firstCollectorConfig,
				secondNginxId: secondCollectorConfig,
			},
		},
		{
			testName: "NginxInstanceRemovedAndNewOneAdded",
			message:  core.NewMessage(core.NginxDetailProcUpdate, []core.Process{}),
			processes: []core.Process{
				{
					Name:     secondNginxPid,
					IsMaster: true,
				},
			},
			expectedNumberOfCollectors: 1,
			expectedCollectorConfigMap: map[string]*metrics.NginxCollectorConfig{
				secondNginxId: secondCollectorConfig,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			env := tutils.NewMockEnvironment()

			env.Mock.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
				Hostname: "test-host",
			})
			env.On("Processes", mock.Anything).Return([]core.Process{
				{
					Name:     firstNginxPid,
					IsMaster: true,
				},
			}).Once()

			metricsPlugin := NewMetrics(config, env, binary)
			metricsPlugin.collectors = []metrics.Collector{
				collectors.NewNginxCollector(config, env, metricsPlugin.collectorConfigsMap[firstNginxId], binary),
			}
			messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{metricsPlugin}, []core.ExtensionPlugin{})
			messagePipe.Run()

			// Update the nginx processes seen
			env.Mock.On("Processes", mock.Anything).Return(tc.processes).Once()

			metricsPlugin.Process(tc.message)

			assert.Equal(t, tc.expectedNumberOfCollectors, len(metricsPlugin.collectors))
			assert.Equal(t, tc.expectedCollectorConfigMap, metricsPlugin.collectorConfigsMap)

			metricsPlugin.Close()
		})
	}
}

func TestMetrics_Process_AgentConfigChanged(t *testing.T) {
	testCases := []struct {
		testName         string
		config           *config.Config
		expUpdatedConfig *config.Config
		updatedTags      bool
	}{
		{
			testName: "ValuesToUpdate",
			config: &config.Config{
				ClientID: "12345",
				Tags:     tutils.InitialConfTags,
				AgentMetrics: config.AgentMetrics{
					BulkSize:           100,
					ReportInterval:     10,
					CollectionInterval: 1,
					Mode:               "aggregated",
				},
			},
			expUpdatedConfig: &config.Config{
				ClientID: "12345",
				Tags:     updateTags,
				AgentMetrics: config.AgentMetrics{
					BulkSize:           100,
					ReportInterval:     10000,
					CollectionInterval: 10,
					Mode:               "streaming",
				},
			},
			updatedTags: true,
		},
		{
			testName: "NoValuesToUpate",
			config: &config.Config{
				ClientID: "12345",
				Tags:     tutils.InitialConfTags,
				AgentMetrics: config.AgentMetrics{
					BulkSize:           100,
					ReportInterval:     10000,
					CollectionInterval: 10,
					Mode:               "streaming",
				},
				Features: config.Defaults.Features,
			},
			expUpdatedConfig: &config.Config{
				ClientID: "12345",
				Tags:     tutils.InitialConfTags,
				AgentMetrics: config.AgentMetrics{
					BulkSize:           100,
					ReportInterval:     10000,
					CollectionInterval: 10,
					Mode:               "aggregated",
				},
				Features: config.Defaults.Features,
			},
			updatedTags: false,
		},
	}

	binary := tutils.GetMockNginxBinary()

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create an agent config and initialize Viper config properties
			// based off of it, clean up when done.
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			// Setup metrics and mock pipeline
			metricsPlugin := NewMetrics(tc.config, tutils.GetMockEnvWithProcess(), binary)

			messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{metricsPlugin}, []core.ExtensionPlugin{})

			messagePipe.Run()

			// Make sure tags are set properly before updating
			sort.Strings(metricsPlugin.conf.Tags)
			assert.Equal(t, tutils.InitialConfTags, metricsPlugin.conf.Tags)

			// Attempt update & check results
			updated, err := config.UpdateAgentConfig("12345", tc.expUpdatedConfig.Tags, tc.expUpdatedConfig.Features)
			assert.Nil(t, err)
			assert.Equal(t, updated, tc.updatedTags)

			// Create message that should trigger a sync agent config call
			msg := core.NewMessage(core.AgentConfigChanged, "")
			metricsPlugin.Process(msg)

			// Check that the config was properly updated
			sort.Strings(tc.expUpdatedConfig.Tags)
			assert.Equal(t, tc.expUpdatedConfig.Tags, metricsPlugin.conf.Tags)

			metricsPlugin.Close()
		})
	}
}

func TestMetrics_Process_RegistrationCompleted(t *testing.T) {
	env := tutils.GetMockEnvWithProcess()
	env.On("IsContainer").Return(false)

	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), env, tutils.GetMockNginxBinary())
	pluginUnderTest.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

	assert.True(t, pluginUnderTest.registrationComplete.Load())
}

func TestMetrics_Process_AgentCollectorsUpdate(t *testing.T) {
	env := tutils.GetMockEnvWithProcess()
	env.On("IsContainer").Return(false)

	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), env, tutils.GetMockNginxBinary())
	pluginUnderTest.Process(core.NewMessage(core.AgentCollectorsUpdate, nil))

	assert.True(t, pluginUnderTest.collectorsUpdate.Load())
}

func TestMetrics_Process_NginxPluginConfigured(t *testing.T) {
	env := tutils.GetMockEnvWithHostAndProcess()
	env.On("IsContainer").Return(false)

	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), env, tutils.GetMockNginxBinary())
	pluginUnderTest.Process(core.NewMessage(core.NginxPluginConfigured, nil))

	assert.GreaterOrEqual(t, len(pluginUnderTest.collectors), 2)
}

func TestMetrics_Process_NginxStatusAPIUpdate_AgentConfigChanged(t *testing.T) {
	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsFromProcess", mock.Anything).Return(detailsMap[secondNginxPid])

	env := tutils.GetMockEnvWithHostAndProcess()
	env.On("IsContainer").Return(false)

	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), env, binary)

	pluginUnderTest.Process(core.NewMessage(core.NginxPluginConfigured, nil))
	conf := pluginUnderTest.collectorConfigsMap[secondNginxId]
	assert.Equal(t, detailsMap[secondNginxPid].ConfPath, conf.ConfPath)

	detailsMap[secondNginxPid].ConfPath = "/something/new/1"

	pluginUnderTest.Process(core.NewMessage(core.NginxStatusAPIUpdate, nil))

	conf = pluginUnderTest.collectorConfigsMap[secondNginxId]
	assert.Equal(t, detailsMap[secondNginxPid].ConfPath, conf.ConfPath)

	detailsMap[secondNginxPid].ConfPath = "/something/new/2"

	pluginUnderTest.Process(core.NewMessage(core.AgentConfigChanged, nil))

	conf = pluginUnderTest.collectorConfigsMap[secondNginxId]
	assert.Equal(t, detailsMap[secondNginxPid].ConfPath, conf.ConfPath)
}

func TestMetrics_Info(t *testing.T) {
	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
	assert.Equal(t, "Metrics", pluginUnderTest.Info().Name())
}

func TestMetrics_Subscriptions(t *testing.T) {
	subs := []string{
		core.RegistrationCompletedTopic,
		core.AgentCollectorsUpdate,
		core.AgentConfigChanged,
		core.NginxStatusAPIUpdate,
		core.NginxPluginConfigured,
		core.NginxDetailProcUpdate,
	}
	pluginUnderTest := NewMetrics(tutils.GetMockAgentConfig(), tutils.GetMockEnvWithProcess(), tutils.GetMockNginxBinary())
	assert.Equal(t, subs, pluginUnderTest.Subscriptions())
}
