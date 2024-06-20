// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/internal/datasource/host/hostfakes"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

func TestResourceService_AddInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		instanceList []*v1.Instance
		resource     *v1.Resource
	}{
		{
			name: "Test 1: Add One Instance",
			instanceList: []*v1.Instance{
				protos.GetNginxOssInstance([]string{}),
			},
			resource: protos.GetHostResource(),
		},
		{
			name: "Test 2: Add Multiple Instance",
			instanceList: []*v1.Instance{
				protos.GetNginxOssInstance([]string{}),
				protos.GetNginxPlusInstance([]string{}),
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
					protos.GetNginxPlusInstance([]string{}),
				},
				Info: protos.GetHostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resource := resourceService.AddInstances(test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_UpdateInstance(t *testing.T) {
	ctx := context.Background()

	updatedInstance := &v1.Instance{
		InstanceConfig: protos.GetNginxOssInstance([]string{}).GetInstanceConfig(),
		InstanceMeta:   protos.GetNginxOssInstance([]string{}).GetInstanceMeta(),
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  56789,
			BinaryPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetBinaryPath(),
			ConfigPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetConfigPath(),
			Details:    protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetDetails(),
		},
	}

	tests := []struct {
		name         string
		instanceList []*v1.Instance
		resource     *v1.Resource
	}{
		{
			name: "Test 1: Update Instances",
			instanceList: []*v1.Instance{
				updatedInstance,
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					updatedInstance,
				},
				Info: protos.GetHostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = []*v1.Instance{protos.GetNginxOssInstance([]string{})}
			resource := resourceService.UpdateInstances(test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_DeleteInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		instanceList []*v1.Instance
		resource     *v1.Resource
		err          error
	}{
		{
			name: "Test 1: Update Instances",
			instanceList: []*v1.Instance{
				protos.GetNginxPlusInstance([]string{}),
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
				Info: protos.GetHostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = []*v1.Instance{
				protos.GetNginxOssInstance([]string{}),
				protos.GetNginxPlusInstance([]string{}),
			}
			resource := resourceService.DeleteInstances(test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_GetResource(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		isContainer      bool
		expectedResource *v1.Resource
	}{
		{
			isContainer:      true,
			expectedResource: protos.GetContainerizedResource(),
		},
		{
			isContainer:      false,
			expectedResource: protos.GetHostResource(),
		},
	}
	for _, tc := range testCases {
		mockInfo := &hostfakes.FakeInfoInterface{}
		if tc.isContainer {
			mockInfo.ContainerInfoReturns(
				&v1.Resource_ContainerInfo{
					ContainerInfo: tc.expectedResource.GetContainerInfo(),
				},
			)
		} else {
			mockInfo.HostInfoReturns(
				&v1.Resource_HostInfo{
					HostInfo: tc.expectedResource.GetHostInfo(),
				},
			)
		}

		mockInfo.IsContainerReturns(tc.isContainer)

		resourceService := NewResourceService(ctx, types.AgentConfig())
		resourceService.info = mockInfo
		resourceService.resource = tc.expectedResource

		resourceService.updateResourceInfo(ctx)
		assert.Equal(t, tc.expectedResource.GetResourceId(), resourceService.resource.GetResourceId())
		assert.Empty(t, resourceService.resource.GetInstances())

		if tc.isContainer {
			assert.Equal(t, tc.expectedResource.GetContainerInfo(), resourceService.resource.GetContainerInfo())
		} else {
			assert.Equal(t, tc.expectedResource.GetHostInfo(), resourceService.resource.GetHostInfo())
		}
	}
}

func TestResourceService_Apply(t *testing.T) {
	ctx := context.Background()

	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		name             string
		out              *bytes.Buffer
		errorLogs        string
		errorLogContents string
		killErr          error
		cmdError         error
		expected         error
	}{
		{
			name:             "Test 1: Successful reload",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: "",
			killErr:          nil,
			cmdError:         nil,
			expected:         nil,
		},
		{
			name:             "Test 2: Successful reload - unknown error log location",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: "",
			killErr:          nil,
			cmdError:         nil,
			expected:         nil,
		},
		{
			name:     "Test 3: Successful reload - no error logs",
			out:      bytes.NewBufferString(""),
			killErr:  nil,
			cmdError: nil,
			expected: nil,
		},
		{
			name:     "Test 4: Failed reload",
			out:      bytes.NewBufferString(""),
			killErr:  errors.New("error reloading"),
			cmdError: nil,
			expected: fmt.Errorf("failed to reload NGINX %w", errors.New("error reloading")),
		},
		{
			name:             "Test 5: Failed reload due to error in error logs",
			out:              bytes.NewBufferString(""),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: errorLogLine,
			cmdError:         nil,
			killErr:          nil,
			expected:         errors.Join(fmt.Errorf(errorLogLine)),
		},
		{
			name:             "Test 6: Failed validating config",
			out:              bytes.NewBufferString("nginx [emerg]"),
			errorLogs:        errorLogFile.Name(),
			errorLogContents: "",
			cmdError:         nil,
			killErr:          nil,
			expected: fmt.Errorf("failed validating config %w",
				errors.New("error running nginx -t -c:\nnginx [emerg]")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.KillProcessReturns(test.killErr)
			mockExec.RunCmdReturns(test.out, test.cmdError)

			instanceOp := NewInstanceOperator()
			instanceOp.executer = mockExec

			logTailOperator := NewLogTailerOperator(types.AgentConfig())

			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceOpMap := make(map[string]instanceOperator)
			resourceOpMap[protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()] = instanceOp
			resourceService.instanceOperators = resourceOpMap
			instance := protos.GetNginxOssInstance([]string{})
			instance.GetInstanceRuntime().GetNginxRuntimeInfo().ErrorLogs = []string{test.errorLogs}
			instances := []*v1.Instance{
				instance,
			}
			resourceService.resource.Instances = instances

			resourceService.logTailer = logTailOperator

			var wg sync.WaitGroup
			wg.Add(1)
			go func(expected error) {
				defer wg.Done()
				reloadError := resourceService.Apply(ctx,
					protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId())
				assert.Equal(t, expected, reloadError)
			}(test.expected)

			time.Sleep(200 * time.Millisecond)

			if test.errorLogContents != "" {
				_, err := errorLogFile.WriteString(test.errorLogContents)
				require.NoError(t, err, "Error writing data to error log file")
			}

			wg.Wait()
		})
	}
}
