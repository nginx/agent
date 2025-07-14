// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/nginx/agent/v3/internal/watcher/health/healthfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestHealthWatcherService_AddHealthWatcher(t *testing.T) {
	agentConfig := types.AgentConfig()
	instance := protos.NginxOssInstance([]string{})

	tests := []struct {
		name        string
		instances   []*mpi.Instance
		numWatchers int
	}{
		{
			name: "Test 1: NGINX Instance",
			instances: []*mpi.Instance{
				instance,
			},
			numWatchers: 1,
		},
		{
			name: "Test 2: Not Supported Instance",
			instances: []*mpi.Instance{
				protos.UnsupportedInstance(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			healthWatcher := NewHealthWatcherService(agentConfig)
			healthWatcher.AddHealthWatcher(test.instances)

			if test.numWatchers == 1 {
				assert.Len(t, healthWatcher.watchers, 1)
				assert.NotNil(t, healthWatcher.watchers[instance.GetInstanceMeta().GetInstanceId()])
				assert.NotNil(t, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])
			} else {
				assert.Empty(t, healthWatcher.watchers)
			}
		})
	}
}

func TestHealthWatcherService_DeleteHealthWatcher(t *testing.T) {
	agentConfig := types.AgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	instance := protos.NginxOssInstance([]string{})

	instances := []*mpi.Instance{instance}
	healthWatcher.AddHealthWatcher(instances)
	assert.Len(t, healthWatcher.watchers, 1)

	healthWatcher.DeleteHealthWatcher(instances)
	assert.Empty(t, healthWatcher.watchers)
	assert.Nil(t, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])
}

func TestHealthWatcherService_UpdateHealthWatcher(t *testing.T) {
	agentConfig := types.AgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	instance := protos.NginxOssInstance([]string{})
	updatedInstance := protos.NginxPlusInstance([]string{})
	updatedInstance.GetInstanceMeta().InstanceId = instance.GetInstanceMeta().GetInstanceId()

	instances := []*mpi.Instance{instance}
	healthWatcher.AddHealthWatcher(instances)
	assert.Equal(t, instance, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])

	healthWatcher.UpdateHealthWatcher([]*mpi.Instance{updatedInstance})

	assert.Equal(t, updatedInstance, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])
}

func TestHealthWatcherService_health(t *testing.T) {
	ctx := context.Background()
	agentConfig := types.AgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	ossInstance := protos.NginxOssInstance([]string{})
	plusInstance := protos.NginxPlusInstance([]string{})
	unspecifiedInstance := protos.UnsupportedInstance()
	watchers := make(map[string]healthWatcherOperator)

	fakeOSSHealthOp := healthfakes.FakeHealthWatcherOperator{}
	fakeOSSHealthOp.HealthReturns(protos.HealthyInstanceHealth(), nil)

	fakePlusHealthOp := healthfakes.FakeHealthWatcherOperator{}
	fakePlusHealthOp.HealthReturns(protos.UnhealthyInstanceHealth(), nil)

	fakeUnspecifiedHealthOp := healthfakes.FakeHealthWatcherOperator{}
	fakeUnspecifiedHealthOp.HealthReturns(nil, errors.New("unable to determine health"))

	watchers[plusInstance.GetInstanceMeta().GetInstanceId()] = &fakePlusHealthOp
	watchers[ossInstance.GetInstanceMeta().GetInstanceId()] = &fakeOSSHealthOp
	watchers[unspecifiedInstance.GetInstanceMeta().GetInstanceId()] = &fakeUnspecifiedHealthOp
	healthWatcher.watchers = watchers

	expected := []*mpi.InstanceHealth{
		protos.HealthyInstanceHealth(),
		protos.UnhealthyInstanceHealth(),
		protos.UnspecifiedInstanceHealth(),
	}

	tests := []struct {
		cache        map[string]*mpi.InstanceHealth
		name         string
		isHealthDiff bool
	}{
		{
			name: "Test 1: Status Changed",
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
				},
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: true,
		},
		{
			name: "Test 2: Status Not Changed",
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId():         protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId():        protos.UnhealthyInstanceHealth(),
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: false,
		},
		{
			name: "Test 3: Less Instances",
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           ossInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
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

func TestHealthWatcherService_compareCache(t *testing.T) {
	ossInstance := protos.NginxOssInstance([]string{})
	plusInstance := protos.NginxPlusInstance([]string{})
	healthCache := map[string]*mpi.InstanceHealth{
		ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
		plusInstance.GetInstanceMeta().GetInstanceId(): {
			InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
			InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
		},
	}

	healths := []*mpi.InstanceHealth{
		protos.HealthyInstanceHealth(),
	}

	tests := []struct {
		name           string
		expectedCache  map[string]*mpi.InstanceHealth
		instances      map[string]*mpi.Instance
		expectedHealth []*mpi.InstanceHealth
	}{
		{
			name: "Test 1: Instance was deleted",
			expectedHealth: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
				{
					InstanceId: plusInstance.GetInstanceMeta().GetInstanceId(),
					Description: fmt.Sprintf("instance %s not found", plusInstance.
						GetInstanceMeta().GetInstanceId()),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
			},
			expectedCache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
			},
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId(): ossInstance,
			},
		},
		{
			name: "Test 2: No change to instance list",
			expectedHealth: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
			},
			expectedCache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
			},
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId(): ossInstance,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			agentConfig := types.AgentConfig()
			healthWatcher := NewHealthWatcherService(agentConfig)
			healthWatcher.cache = healthCache
			healthWatcher.instances = test.instances

			result := healthWatcher.compareCache(healths)

			assert.Equal(t, test.expectedHealth, result)
			assert.Equal(t, test.expectedCache, healthWatcher.cache)
		})
	}
}

func TestHealthWatcherService_GetInstancesHealth(t *testing.T) {
	ossInstance := protos.NginxOssInstance([]string{})
	plusInstance := protos.NginxPlusInstance([]string{})
	ossInstanceHealth := protos.HealthyInstanceHealth()
	plusInstanceHealth := protos.UnhealthyInstanceHealth()

	healthCache := map[string]*mpi.InstanceHealth{
		ossInstance.GetInstanceMeta().GetInstanceId():  ossInstanceHealth,
		plusInstance.GetInstanceMeta().GetInstanceId(): plusInstanceHealth,
	}

	expectedInstancesHealth := []*mpi.InstanceHealth{
		ossInstanceHealth,
		plusInstanceHealth,
	}
	agentConfig := types.AgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	healthWatcher.cache = healthCache

	result := healthWatcher.InstancesHealth()

	assert.ElementsMatch(t, expectedInstancesHealth, result)
}
