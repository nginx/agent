// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/internal/datasource/host/hostfakes"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
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
				protos.GetAgentInstance(1, types.GetAgentConfig()),
				protos.GetNginxOssInstance([]string{}),
			},
			resource: protos.GetHostResource(),
		},
		{
			name: "Test 2: Add Multiple Instance",
			instanceList: []*v1.Instance{
				protos.GetAgentInstance(1, types.GetAgentConfig()),
				protos.GetNginxOssInstance([]string{}),
				protos.GetNginxPlusInstance([]string{}),
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.GetAgentInstance(1, types.GetAgentConfig()),
					protos.GetNginxOssInstance([]string{}),
					protos.GetNginxPlusInstance([]string{}),
				},
				Info: protos.GetHostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx)
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
			resourceService := NewResourceService(ctx)
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
			resourceService := NewResourceService(ctx)
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

		resourceService := NewResourceService(ctx)
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
