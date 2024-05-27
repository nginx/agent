// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"
	testModel "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceWatcherService_checkForUpdates(t *testing.T) {
	ctx := context.Background()

	nginxConfigContext := testModel.GetConfigContext()

	fakeProcessWatcher := &watcherfakes.FakeProcessWatcherOperator{}
	fakeProcessWatcher.ProcessesReturns([]*model.Process{}, nil)

	fakeProcessParser := &watcherfakes.FakeProcessParser{}
	fakeProcessParser.ParseReturns(map[string]*v1.Instance{
		protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.
			GetNginxOssInstance([]string{}),
	})

	fakeNginxConfigParser := &watcherfakes.FakeNginxConfigParser{}
	fakeNginxConfigParser.ParseReturns(nginxConfigContext, nil)

	instanceWatcherService := NewInstanceWatcherService(types.GetAgentConfig())
	instanceWatcherService.processOperator = fakeProcessWatcher
	instanceWatcherService.processParsers = []processParser{fakeProcessParser}
	instanceWatcherService.nginxConfigParser = fakeNginxConfigParser

	instanceUpdatesChannel := make(chan InstanceUpdatesMessage, 1)
	nginxConfigContextChannel := make(chan NginxConfigContextMessage, 1)

	instanceWatcherService.checkForUpdates(ctx, instanceUpdatesChannel, nginxConfigContextChannel)

	instanceUpdatesMessage := <-instanceUpdatesChannel
	assert.Len(t, instanceUpdatesMessage.instanceUpdates.newInstances, 2)
	assert.Empty(t, instanceUpdatesMessage.instanceUpdates.deletedInstances)

	nginxConfigContextMessage := <-nginxConfigContextChannel
	assert.Equal(t, nginxConfigContext, nginxConfigContextMessage.nginxConfigContext)
}

func TestInstanceWatcherService_instanceUpdates(t *testing.T) {
	ctx := context.Background()
	processID := int32(123)

	agentInstance := protos.GetAgentInstance(processID, types.GetAgentConfig())

	tests := []struct {
		name                    string
		oldInstances            []*v1.Instance
		parsedInstances         map[string]*v1.Instance
		expectedInstanceUpdates InstanceUpdates
	}{
		{
			name:                    "Test 1: No updates",
			oldInstances:            []*v1.Instance{agentInstance},
			parsedInstances:         make(map[string]*v1.Instance),
			expectedInstanceUpdates: InstanceUpdates{},
		},
		{
			name:         "Test 2: New instance",
			oldInstances: []*v1.Instance{agentInstance},
			parsedInstances: map[string]*v1.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
				protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.GetNginxOssInstance(
					[]string{}),
			},
			expectedInstanceUpdates: InstanceUpdates{
				newInstances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
			},
		},
		{
			name: "Test 3: Deleted instance",
			oldInstances: []*v1.Instance{
				agentInstance,
				protos.GetNginxOssInstance([]string{}),
			},
			parsedInstances: make(map[string]*v1.Instance),
			expectedInstanceUpdates: InstanceUpdates{
				deletedInstances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeProcessWatcher := &watcherfakes.FakeProcessWatcherOperator{}
			fakeProcessWatcher.ProcessesReturns([]*model.Process{}, nil)

			fakeProcessParser := &watcherfakes.FakeProcessParser{}
			fakeProcessParser.ParseReturns(test.parsedInstances)

			fakeExec := &execfakes.FakeExecInterface{}
			fakeExec.ExecutableReturns(defaultAgentPath, nil)
			fakeExec.ProcessIDReturns(processID)

			instanceWatcherService := NewInstanceWatcherService(types.GetAgentConfig())
			instanceWatcherService.processOperator = fakeProcessWatcher
			instanceWatcherService.processParsers = []processParser{fakeProcessParser}
			instanceWatcherService.instanceCache = test.oldInstances
			instanceWatcherService.executer = fakeExec

			instanceUpdates, err := instanceWatcherService.instanceUpdates(ctx)

			require.NoError(tt, err)
			assert.Equal(tt, test.expectedInstanceUpdates, instanceUpdates)
		})
	}
}

func TestInstanceWatcherService_updateNginxInstanceRuntime(t *testing.T) {
	instanceWatcherService := NewInstanceWatcherService(types.GetAgentConfig())

	nginxOSSConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		StubStatus: "http://127.0.0.1:8081/api",
	}

	nginxPlusConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		PlusAPI: "http://127.0.0.1:8081/api",
	}

	tests := []struct {
		name               string
		nginxConfigContext *model.NginxConfigContext
		instance           *v1.Instance
	}{
		{
			name:               "Test 1: OSS Instance",
			nginxConfigContext: nginxOSSConfigContext,
			instance:           protos.GetNginxOssInstance([]string{}),
		},
		{
			name:               "Test 2: Plus Instance",
			nginxConfigContext: nginxPlusConfigContext,
			instance:           protos.GetNginxPlusInstance([]string{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			instanceWatcherService.updateNginxInstanceRuntime(test.instance, test.nginxConfigContext)
			if test.name == "Test 2: Plus Instance" {
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetStubStatus())
				assert.Equal(t, test.nginxConfigContext.PlusAPI, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi())
			} else {
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetStubStatus())
			}
		})
	}
}
