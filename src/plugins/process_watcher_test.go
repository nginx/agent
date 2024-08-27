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

	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestProcessWatcher_getProcUpdates(t *testing.T) {
	tests := []struct {
		name                string
		seenMasterProcs     map[int32]*core.Process
		seenWorkerProcs     map[int32]*core.Process
		seenNginxDetails    map[int32]*proto.NginxDetails
		nginxDetails        []*proto.NginxDetails
		nginxProcs          []*core.Process
		expectedProcUpdates map[string]string
		expectedMasterPids  []int32
		expectedWorkerPids  []int32
	}{
		{
			name:             "nginx startup",
			seenMasterProcs:  map[int32]*core.Process{},
			seenWorkerProcs:  map[int32]*core.Process{},
			seenNginxDetails: map[int32]*proto.NginxDetails{},
			nginxDetails: []*proto.NginxDetails{
				{
					NginxId: "1", Version: "21", ConfPath: "/etc/yo", ProcessId: "1", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "2", Version: "21", ConfPath: "/etc/yo", ProcessId: "2", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "3", Version: "21", ConfPath: "/etc/yo", ProcessId: "3", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
			},
			nginxProcs: []*core.Process{
				tutils.GetProcesses()[0],
				tutils.GetProcesses()[1],
				tutils.GetProcesses()[2],
			},
			expectedProcUpdates: map[string]string{
				"1": "nginx.master.created",
				"2": "nginx.worker.created",
				"3": "nginx.worker.created",
			},
			expectedMasterPids: []int32{1},
			expectedWorkerPids: []int32{2, 3},
		},
		{
			name: "nginx reload",
			seenMasterProcs: map[int32]*core.Process{
				1: tutils.GetProcesses()[0],
			},
			seenWorkerProcs: map[int32]*core.Process{
				2: tutils.GetProcesses()[1],
				3: tutils.GetProcesses()[2],
			},
			seenNginxDetails: map[int32]*proto.NginxDetails{
				1: {ProcessId: "1"},
				2: {ProcessId: "2"},
				3: {ProcessId: "3"},
			},
			nginxDetails: []*proto.NginxDetails{
				{
					NginxId: "4", Version: "21", ConfPath: "/etc/yo", ProcessId: "4", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "5", Version: "21", ConfPath: "/etc/yo", ProcessId: "5", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
			},
			nginxProcs: []*core.Process{
				tutils.GetProcesses()[0],
				{Pid: 4, ParentPid: 1, Name: "worker-1", IsMaster: false},
				{Pid: 5, ParentPid: 1, Name: "worker-2", IsMaster: false},
			},
			expectedProcUpdates: map[string]string{
				"4": "nginx.worker.created",
				"5": "nginx.worker.created",
				"2": "nginx.worker.killed",
				"3": "nginx.worker.killed",
			},
			expectedMasterPids: []int32{1},
			expectedWorkerPids: []int32{4, 5},
		},
		{
			name: "nginx stop && nginx start",
			seenMasterProcs: map[int32]*core.Process{
				1: tutils.GetProcesses()[0],
			},
			seenWorkerProcs: map[int32]*core.Process{
				2: tutils.GetProcesses()[1],
				3: tutils.GetProcesses()[2],
			},
			seenNginxDetails: map[int32]*proto.NginxDetails{
				1: {ProcessId: "1"},
				2: {ProcessId: "2"},
				3: {ProcessId: "3"},
			},
			nginxProcs: []*core.Process{
				{Pid: 6, Name: "master", IsMaster: true},
				{Pid: 7, ParentPid: 6, Name: "worker-1", IsMaster: false},
				{Pid: 8, ParentPid: 6, Name: "worker-2", IsMaster: false},
			},
			nginxDetails: []*proto.NginxDetails{
				{
					NginxId: "6", Version: "21", ConfPath: "/etc/yo", ProcessId: "6", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "7", Version: "21", ConfPath: "/etc/yo", ProcessId: "7", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "8", Version: "21", ConfPath: "/etc/yo", ProcessId: "8", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "1", Version: "21", ConfPath: "/etc/yo", ProcessId: "1", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "2", Version: "21", ConfPath: "/etc/yo", ProcessId: "2", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
				{
					NginxId: "3", Version: "21", ConfPath: "/etc/yo", ProcessId: "3", StartTime: 1238043824,
					BuiltFromSource: false,
					LoadableModules: []string{},
					RuntimeModules:  []string{},
					Plus:            &proto.NginxPlusMetaData{Enabled: true},
					ConfigureArgs:   []string{},
				},
			},
			expectedProcUpdates: map[string]string{
				"6": "nginx.master.created",
				"7": "nginx.worker.created",
				"8": "nginx.worker.created",
				"1": "nginx.master.killed",
				"2": "nginx.worker.killed",
				"3": "nginx.worker.killed",
			},
			expectedMasterPids: []int32{6},
			expectedWorkerPids: []int32{7, 8},
		},
		{
			name: "nginx stop",
			seenMasterProcs: map[int32]*core.Process{
				1: tutils.GetProcesses()[0],
			},
			seenWorkerProcs: map[int32]*core.Process{
				2: tutils.GetProcesses()[1],
				3: tutils.GetProcesses()[2],
			},
			seenNginxDetails: map[int32]*proto.NginxDetails{
				1: {ProcessId: "1"},
				2: {ProcessId: "2"},
				3: {ProcessId: "3"},
			},
			nginxProcs: []*core.Process{},
			expectedProcUpdates: map[string]string{
				"1": "nginx.master.killed",
				"2": "nginx.worker.killed",
				"3": "nginx.worker.killed",
			},
			expectedMasterPids: []int32{},
			expectedWorkerPids: []int32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tutils.NewMockEnvironment()
			binary := &tutils.MockNginxBinary{}

			for _, nginxDetail := range tt.nginxDetails {
				binary.On("GetNginxDetailsFromProcess", mock.Anything).Return(nginxDetail).Once()
			}

			pw := NewProcessWatcher(env, binary, tt.nginxProcs, &config.Config{})
			pw.seenMasterProcs = tt.seenMasterProcs
			pw.seenWorkerProcs = tt.seenWorkerProcs
			pw.nginxDetails = tt.seenNginxDetails

			procUpdates, masterProcs, workerProcs := pw.getProcUpdates(tt.nginxProcs)

			for _, procUpdate := range procUpdates {
				if expectedTopic, ok := tt.expectedProcUpdates[procUpdate.Data().(*proto.NginxDetails).ProcessId]; !ok {
					assert.Fail(t, "Missing expected pid")
				} else {
					assert.Equal(t, expectedTopic, procUpdate.Topic())
				}
			}

			for pid := range masterProcs {
				assert.Contains(t, tt.expectedMasterPids, pid)
			}
			for pid := range workerProcs {
				assert.Contains(t, tt.expectedWorkerPids, pid)
			}
		})
	}
}

func TestProcessWatcher_Process(t *testing.T) {
	env := tutils.GetMockEnv()
	pluginUnderTest := NewProcessWatcher(env, tutils.GetMockNginxBinary(), tutils.GetProcesses(), &config.Config{})

	ctx, cancel := context.WithCancel(context.TODO())
	messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

	pluginUnderTest.Init(messagePipe)
	messagePipe.Run()

	msgTopics := []string{core.NginxDetailProcUpdate}
	messages := messagePipe.GetMessages()

	for idx, msg := range messages {
		if msgTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), msgTopics[idx])
		}
	}

	cancel()

	pluginUnderTest.Close()
}

func TestProcessWatcher_Subscription(t *testing.T) {
	pluginUnderTest := NewProcessWatcher(nil, nil, nil, &config.Config{})

	assert.Equal(t, []string{}, pluginUnderTest.Subscriptions())
}

func TestProcessWatcher_Info(t *testing.T) {
	pluginUnderTest := NewProcessWatcher(nil, nil, nil, &config.Config{})

	assert.Equal(t, "process-watcher", pluginUnderTest.Info().Name())
}
