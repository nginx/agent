// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/internal/watcher/instance/instancefakes"
	"github.com/nginx/agent/v3/internal/watcher/process/processfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/internal/model"
	testModel "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceWatcherService_checkForUpdates(t *testing.T) {
	ctx := context.Background()

	nginxConfigContext := testModel.GetConfigContext()

	fakeProcessWatcher := &processfakes.FakeProcessOperatorInterface{}
	fakeProcessWatcher.ProcessesReturns([]*model.Process{}, nil)

	fakeProcessParser := &instancefakes.FakeProcessParser{}
	fakeProcessParser.ParseReturns(map[string]*mpi.Instance{
		protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.
			GetNginxOssInstance([]string{}),
	})

	fakeNginxConfigParser := &instancefakes.FakeNginxConfigParser{}
	fakeNginxConfigParser.ParseReturns(nginxConfigContext, nil)
	instanceUpdatesChannel := make(chan InstanceUpdatesMessage, 1)
	nginxConfigContextChannel := make(chan NginxConfigContextMessage, 1)

	instanceWatcherService := NewInstanceWatcherService(types.AgentConfig())
	instanceWatcherService.processOperator = fakeProcessWatcher
	instanceWatcherService.processParsers = []processParser{fakeProcessParser}
	instanceWatcherService.nginxConfigParser = fakeNginxConfigParser
	instanceWatcherService.instancesChannel = instanceUpdatesChannel
	instanceWatcherService.nginxConfigContextChannel = nginxConfigContextChannel

	instanceWatcherService.checkForUpdates(ctx)

	instanceUpdatesMessage := <-instanceUpdatesChannel
	assert.Len(t, instanceUpdatesMessage.InstanceUpdates.NewInstances, 2)
	assert.Empty(t, instanceUpdatesMessage.InstanceUpdates.DeletedInstances)

	nginxConfigContextMessage := <-nginxConfigContextChannel
	assert.Equal(t, nginxConfigContext, nginxConfigContextMessage.NginxConfigContext)
}

func TestInstanceWatcherService_instanceUpdates(t *testing.T) {
	ctx := context.Background()
	processID := int32(123)

	agentInstance := protos.GetAgentInstance(processID, types.AgentConfig())
	nginxInstance := protos.GetNginxOssInstance([]string{})
	nginxInstanceWithDifferentPID := protos.GetNginxOssInstance([]string{})
	nginxInstanceWithDifferentPID.GetInstanceRuntime().ProcessId = 3526

	tests := []struct {
		name                    string
		oldInstances            map[string]*mpi.Instance
		parsedInstances         map[string]*mpi.Instance
		expectedInstanceUpdates InstanceUpdates
	}{
		{
			name: "Test 1: No updates",
			oldInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
			},
			parsedInstances:         make(map[string]*mpi.Instance),
			expectedInstanceUpdates: InstanceUpdates{},
		},
		{
			name: "Test 2: New instance",
			oldInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
			},
			parsedInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
				nginxInstance.GetInstanceMeta().GetInstanceId(): nginxInstance,
			},
			expectedInstanceUpdates: InstanceUpdates{
				NewInstances: []*mpi.Instance{
					nginxInstance,
				},
			},
		},
		{
			name: "Test 3: Updated instance",
			oldInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId():                 agentInstance,
				nginxInstanceWithDifferentPID.GetInstanceMeta().GetInstanceId(): nginxInstanceWithDifferentPID,
			},
			parsedInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
				nginxInstance.GetInstanceMeta().GetInstanceId(): nginxInstance,
			},
			expectedInstanceUpdates: InstanceUpdates{
				UpdatedInstances: []*mpi.Instance{
					nginxInstance,
				},
			},
		},
		{
			name: "Test 4: Deleted instance",
			oldInstances: map[string]*mpi.Instance{
				agentInstance.GetInstanceMeta().GetInstanceId(): agentInstance,
				protos.GetNginxOssInstance([]string{}).GetInstanceMeta().
					GetInstanceId(): protos.GetNginxOssInstance([]string{}),
			},
			parsedInstances: make(map[string]*mpi.Instance),
			expectedInstanceUpdates: InstanceUpdates{
				DeletedInstances: []*mpi.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeProcessWatcher := &processfakes.FakeProcessOperatorInterface{}
			fakeProcessWatcher.ProcessesReturns([]*model.Process{}, nil)

			fakeProcessParser := &instancefakes.FakeProcessParser{}
			fakeProcessParser.ParseReturns(test.parsedInstances)

			fakeExec := &execfakes.FakeExecInterface{}
			fakeExec.ExecutableReturns(defaultAgentPath, nil)
			fakeExec.ProcessIDReturns(processID)

			instanceWatcherService := NewInstanceWatcherService(types.AgentConfig())
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
	instanceWatcherService := NewInstanceWatcherService(types.AgentConfig())

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
		nginxConfigContext *model.NginxConfigContext
		instance           *mpi.Instance
		name               string
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

func TestInstanceWatcherService_areInstancesEqual(t *testing.T) {
	tests := []struct {
		oldRuntime     *mpi.InstanceRuntime
		currentRuntime *mpi.InstanceRuntime
		name           string
		expected       bool
	}{
		{
			name: "Test 1: Instances are equal",
			oldRuntime: &mpi.InstanceRuntime{
				ProcessId: 123,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 111,
					},
					{
						ProcessId: 222,
					},
				},
			},
			currentRuntime: &mpi.InstanceRuntime{
				ProcessId: 123,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 222,
					},
					{
						ProcessId: 111,
					},
				},
			},
			expected: true,
		},
		{
			name: "Test 2: Different PIDs",
			oldRuntime: &mpi.InstanceRuntime{
				ProcessId: 123,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 111,
					},
				},
			},
			currentRuntime: &mpi.InstanceRuntime{
				ProcessId: 456,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 111,
					},
				},
			},
			expected: false,
		},
		{
			name: "Test 3: Different children PIDs",
			oldRuntime: &mpi.InstanceRuntime{
				ProcessId: 123,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 111,
					},
					{
						ProcessId: 333,
					},
				},
			},
			currentRuntime: &mpi.InstanceRuntime{
				ProcessId: 123,
				InstanceChildren: []*mpi.InstanceChild{
					{
						ProcessId: 111,
					},
					{
						ProcessId: 222,
					},
				},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			assert.Equal(t, test.expected, areInstancesEqual(test.oldRuntime, test.currentRuntime))
		})
	}
}

func TestInstanceWatcherService_ReparseConfig(t *testing.T) {
	ctx := context.Background()

	nginxConfigContext := testModel.GetConfigContext()
	updateNginxConfigContext := testModel.GetConfigContext()
	updateNginxConfigContext.AccessLogs = []*model.AccessLog{
		{
			Name: "access2.log",
		},
	}

	instance := protos.GetNginxOssInstance([]string{})
	instance.GetInstanceRuntime().GetNginxRuntimeInfo().AccessLogs = []string{"access.logs"}
	instance.GetInstanceRuntime().GetNginxRuntimeInfo().ErrorLogs = []string{"error.log"}

	updatedInstance := protos.GetNginxOssInstance([]string{})
	updatedInstance.GetInstanceRuntime().GetNginxRuntimeInfo().AccessLogs = []string{"access2.log"}
	updatedInstance.GetInstanceRuntime().GetNginxRuntimeInfo().ErrorLogs = []string{"error.log"}

	tests := []struct {
		parseReturns *model.NginxConfigContext
		name         string
	}{
		{
			name:         "Test 1: Config Context Different",
			parseReturns: updateNginxConfigContext,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeNginxConfigParser := &instancefakes.FakeNginxConfigParser{}
			fakeNginxConfigParser.ParseReturns(test.parseReturns, nil)
			instanceUpdatesChannel := make(chan InstanceUpdatesMessage, 1)
			nginxConfigContextChannel := make(chan NginxConfigContextMessage, 1)

			instanceWatcherService := NewInstanceWatcherService(types.AgentConfig())
			instanceWatcherService.nginxConfigParser = fakeNginxConfigParser
			instanceWatcherService.instancesChannel = instanceUpdatesChannel
			instanceWatcherService.nginxConfigContextChannel = nginxConfigContextChannel

			instanceWatcherService.nginxConfigCache = map[string]*model.NginxConfigContext{
				instance.GetInstanceMeta().GetInstanceId(): nginxConfigContext,
			}

			instanceWatcherService.instanceCache = map[string]*mpi.Instance{
				instance.GetInstanceMeta().GetInstanceId(): instance,
			}

			instanceWatcherService.ReparseConfig(ctx, updatedInstance.GetInstanceMeta().GetInstanceId())

			nginxConfigContextMessage := <-nginxConfigContextChannel
			assert.Equal(t, updateNginxConfigContext, nginxConfigContextMessage.NginxConfigContext)

			instanceUpdatesMessage := <-instanceUpdatesChannel
			assert.Len(t, instanceUpdatesMessage.InstanceUpdates.UpdatedInstances, 1)
			assert.Equal(tt, updatedInstance, instanceUpdatesMessage.InstanceUpdates.UpdatedInstances[0])
			assert.Empty(t, instanceUpdatesMessage.InstanceUpdates.DeletedInstances)
		})
	}
}
