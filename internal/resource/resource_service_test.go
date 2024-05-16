// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"fmt"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

func TestResourceService_AddInstance(t *testing.T) {
	tests := []struct {
		name     string
		msg      *bus.Message
		resource *v1.Resource
		err      error
	}{
		{
			name: "Test 1: Add One Instance",
			msg: &bus.Message{
				Topic: bus.NewInstances,
				Data: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
			},
			resource: protos.GetHostResource(),
			err:      nil,
		},
		{
			name: "Test 2: Add Multiple Instance",
			msg: &bus.Message{
				Topic: bus.NewInstances,
				Data: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
					protos.GetNginxPlusInstance([]string{}),
				},
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
					protos.GetNginxPlusInstance([]string{}),
				},
				Info: protos.GetHostResource().GetInfo(),
			},
			err: nil,
		},
		{
			name: "Test 3: Error",
			msg: &bus.Message{
				Topic: bus.NewInstances,
				Data:  nil,
			},
			resource: nil,
			err:      fmt.Errorf("unable to cast message payload to []*v1.Instance, payload, %v", nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService()
			resource, err := resourceService.AddInstance(test.msg)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
			assert.Equal(tt, test.err, err)
		})
	}
}

func TestResourceService_UpdateInstance(t *testing.T) {
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
		name     string
		msg      *bus.Message
		resource *v1.Resource
		err      error
	}{
		{
			name: "Test 1: Update Instances",
			msg: &bus.Message{
				Topic: bus.UpdatedInstances,
				Data: []*v1.Instance{
					updatedInstance,
				},
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					updatedInstance,
				},
				Info: protos.GetHostResource().GetInfo(),
			},
			err: nil,
		},
		{
			name: "Test 2: Error",
			msg: &bus.Message{
				Topic: bus.UpdatedInstances,
				Data:  nil,
			},
			resource: nil,
			err:      fmt.Errorf("unable to cast message payload to []*v1.Instance, payload, %v", nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService()
			resourceService.resource.Instances = []*v1.Instance{protos.GetNginxOssInstance([]string{})}
			resource, err := resourceService.UpdateInstance(test.msg)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
			assert.Equal(tt, test.err, err)
		})
	}
}

func TestResourceService_DeleteInstance(t *testing.T) {
	tests := []struct {
		name     string
		msg      *bus.Message
		resource *v1.Resource
		err      error
	}{
		{
			name: "Test 1: Update Instances",
			msg: &bus.Message{
				Topic: bus.DeletedInstances,
				Data: []*v1.Instance{
					protos.GetNginxPlusInstance([]string{}),
				},
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
				Info: protos.GetHostResource().GetInfo(),
			},
			err: nil,
		},
		{
			name: "Test 2: Error",
			msg: &bus.Message{
				Topic: bus.DeletedInstances,
				Data:  nil,
			},
			resource: nil,
			err:      fmt.Errorf("unable to cast message payload to []*v1.Instance, payload, %v", nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService()
			resourceService.resource.Instances = []*v1.Instance{
				protos.GetNginxOssInstance([]string{}),
				protos.GetNginxPlusInstance([]string{}),
			}
			resource, err := resourceService.DeleteInstance(test.msg)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
			assert.Equal(tt, test.err, err)
		})
	}
}
