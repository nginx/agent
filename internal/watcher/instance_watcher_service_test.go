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
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceWatcherService_Updates(t *testing.T) {
	ctx := context.Background()
	processID := int32(123)

	agentInstance := protos.GetAgentInstance(processID, types.GetAgentConfig())

	tests := []struct {
		name                    string
		oldInstances            []*v1.Instance
		parsedInstances         []*v1.Instance
		expectedInstanceUpdates InstanceUpdates
	}{
		{
			name:                    "Test 1: No updates",
			oldInstances:            []*v1.Instance{agentInstance},
			parsedInstances:         []*v1.Instance{},
			expectedInstanceUpdates: InstanceUpdates{},
		},
		{
			name:         "Test 2: New instance",
			oldInstances: []*v1.Instance{agentInstance},
			parsedInstances: []*v1.Instance{
				agentInstance,
				protos.GetNginxOssInstance([]string{}),
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
			parsedInstances: []*v1.Instance{},
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
			instanceWatcherService.cache = test.oldInstances
			instanceWatcherService.executer = fakeExec

			instanceUpdates, err := instanceWatcherService.updates(ctx)

			require.NoError(tt, err)
			assert.Equal(tt, test.expectedInstanceUpdates, instanceUpdates)
		})
	}
}
