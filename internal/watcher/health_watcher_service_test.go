// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"reflect"
	"testing"

	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestHealthWatcherService_AddHealthWatcher(t *testing.T) {
	agentConfig := types.GetAgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	instance := protos.GetNginxOssInstance([]string{})

	instances := []*v1.Instance{instance}
	healthWatcher.AddHealthWatcher(instances)

	assert.Len(t, healthWatcher.watchers, 1)
	assert.NotNil(t, healthWatcher.watchers[instance.GetInstanceMeta().GetInstanceId()])
}

func TestHealthWatcherService_DeleteHealthWatcher(t *testing.T) {
	agentConfig := types.GetAgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	instance := protos.GetNginxOssInstance([]string{})

	instances := []*v1.Instance{instance}
	healthWatcher.AddHealthWatcher(instances)
	assert.Len(t, healthWatcher.watchers, 1)

	healthWatcher.DeleteHealthWatcher(instances)
	assert.Empty(t, healthWatcher.watchers)
}

func TestHealthWatcherService_health(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.GetAgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	ossInstance := protos.GetNginxOssInstance([]string{})
	plusInstance := protos.GetNginxPlusInstance([]string{})
	watchers := make(map[string]healthWatcherOperator)

	fakeOSSHealthOp := watcherfakes.FakeHealthWatcherOperator{}
	fakeOSSHealthOp.HealthReturns(&v1.InstanceHealth{
		InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
		Description:          "instance is healthy",
		InstanceHealthStatus: 1,
	})

	fakePlusHealthOp := watcherfakes.FakeHealthWatcherOperator{}
	fakePlusHealthOp.HealthReturns(&v1.InstanceHealth{
		InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
		Description:          "instance is unhealthy",
		InstanceHealthStatus: 2,
	})

	watchers[plusInstance.GetInstanceMeta().GetInstanceId()] = &fakePlusHealthOp
	watchers[ossInstance.GetInstanceMeta().GetInstanceId()] = &fakeOSSHealthOp
	healthWatcher.watchers = watchers

	expected := []*v1.InstanceHealth{
		{
			InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
			Description:          "instance is healthy",
			InstanceHealthStatus: 1,
		},
		{
			InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
			Description:          "instance is unhealthy",
			InstanceHealthStatus: 2,
		},
	}

	tests := []struct {
		name         string
		cache        map[string]*v1.InstanceHealth
		isHealthDiff bool
	}{
		{
			name: "Test 1: Status Changed",
			cache: map[string]*v1.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
					Description:          "instance is healthy",
					InstanceHealthStatus: 1,
				},
				plusInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					Description:          "instance is healthy",
					InstanceHealthStatus: 1,
				},
			},
			isHealthDiff: true,
		},
		{
			name: "Test 2: Status Not Changed",
			cache: map[string]*v1.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
					Description:          "instance is healthy",
					InstanceHealthStatus: 1,
				},
				plusInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					Description:          "instance is unhealthy",
					InstanceHealthStatus: 2,
				},
			},
			isHealthDiff: false,
		},
		{
			name: "Test 3: Less Instances",
			cache: map[string]*v1.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
					Description:          "instance is healthy",
					InstanceHealthStatus: 1,
				},
			},
			isHealthDiff: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			healthWatcher.updateCache(test.cache)
			instanceHealth, healthDiff := healthWatcher.health(ctx)
			assert.Equal(t, test.isHealthDiff, healthDiff)

			reflect.DeepEqual(instanceHealth, expected)
		})
	}
}
