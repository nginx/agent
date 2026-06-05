// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/watcher/health/healthfakes"
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			healthWatcher := NewHealthWatcherService(agentConfig)
			assert.NotNil(t, healthWatcher.watcher)
		})
	}
}

func TestHealthWatcherService_UpdateHealthWatcher(t *testing.T) {
	agentConfig := types.AgentConfig()
	healthWatcher := NewHealthWatcherService(agentConfig)
	instance := protos.NginxOssInstance([]string{})

	healthWatcher.instances = map[string]*mpi.Instance{
		instance.GetInstanceMeta().GetInstanceId(): instance,
	}

	updatedInstance := protos.NginxPlusInstance([]string{})
	updatedInstance.GetInstanceMeta().InstanceId = instance.GetInstanceMeta().GetInstanceId()

	assert.Equal(t, instance, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])

	healthWatcher.UpdateHealthWatcher(t.Context(), []*mpi.Instance{updatedInstance})
	assert.Equal(t, updatedInstance, healthWatcher.instances[instance.GetInstanceMeta().GetInstanceId()])
}

func TestHealthWatcherService_health(t *testing.T) {
	ossInstance := protos.NginxOssInstance([]string{})
	plusInstance := protos.NginxPlusInstance([]string{})
	unspecifiedInstance := protos.UnsupportedInstance()

	tests := []struct {
		instances        map[string]*mpi.Instance
		name             string
		cache            map[string]*mpi.InstanceHealth
		updatedInstances []*mpi.InstanceHealth
		isHealthDiff     bool
	}{
		{
			name: "Test 1: NGINX Instance Status Changed",
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId():         ossInstance,
				plusInstance.GetInstanceMeta().GetInstanceId():        plusInstance,
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): unspecifiedInstance,
			},
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
				},
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: true,
			updatedInstances: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
				{
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
				protos.UnspecifiedInstanceHealth(),
			},
		},
		{
			name: "Test 2: NGINX Instance No Status Changed",
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId():         ossInstance,
				plusInstance.GetInstanceMeta().GetInstanceId():        plusInstance,
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): unspecifiedInstance,
			},
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId():         protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId():        protos.UnhealthyInstanceHealth(),
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: false,
			updatedInstances: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
				{
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
				protos.UnspecifiedInstanceHealth(),
			},
		},
		{
			name: "Test 3: Deleted NGINX Instances ",
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId():         ossInstance,
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): unspecifiedInstance,
			},
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId():         protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId():        protos.UnhealthyInstanceHealth(),
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: true,
			updatedInstances: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
				{
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
				protos.UnspecifiedInstanceHealth(),
			},
		},
		{
			name: "Test 4: Added NGINX Instances ",
			instances: map[string]*mpi.Instance{
				ossInstance.GetInstanceMeta().GetInstanceId():         ossInstance,
				plusInstance.GetInstanceMeta().GetInstanceId():        plusInstance,
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): unspecifiedInstance,
			},
			cache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId():         protos.HealthyInstanceHealth(),
				unspecifiedInstance.GetInstanceMeta().GetInstanceId(): protos.UnspecifiedInstanceHealth(),
			},
			isHealthDiff: true,
			updatedInstances: []*mpi.InstanceHealth{
				protos.HealthyInstanceHealth(),
				{
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
				},
				protos.UnspecifiedInstanceHealth(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agentConfig := types.AgentConfig()
			healthWatcher := NewHealthWatcherService(agentConfig)
			fakeHealthWatcher := healthfakes.FakeHealthWatcherOperator{}

			// Dispatch by instance ID so results are independent of map iteration order.
			ossID := ossInstance.GetInstanceMeta().GetInstanceId()
			plusID := plusInstance.GetInstanceMeta().GetInstanceId()
			fakeHealthWatcher.HealthCalls(func(_ context.Context, instance *mpi.Instance) (*mpi.InstanceHealth, error) {
				switch instance.GetInstanceMeta().GetInstanceId() {
				case ossID:
					return protos.HealthyInstanceHealth(), nil
				case plusID:
					return protos.UnhealthyInstanceHealth(), nil
				default:
					return nil, errors.New("unable to determine health")
				}
			})

			healthWatcher.instances = test.instances
			healthWatcher.updateCache(test.cache)
			healthWatcher.watcher = &fakeHealthWatcher
			updatedStatus, isHealthDiff := healthWatcher.health(t.Context())
			assert.Equal(t, test.isHealthDiff, isHealthDiff)
			assert.ElementsMatch(t, test.updatedInstances, updatedStatus)
		})
	}
}

func TestHealthWatcherService_compareCache(t *testing.T) {
	ossInstance := protos.NginxOssInstance([]string{})
	plusInstance := protos.NginxPlusInstance([]string{})

	healths := []*mpi.InstanceHealth{
		protos.HealthyInstanceHealth(),
	}

	tests := []struct {
		name           string
		initialCache   map[string]*mpi.InstanceHealth
		expectedCache  map[string]*mpi.InstanceHealth
		instances      map[string]*mpi.Instance
		expectedHealth []*mpi.InstanceHealth
	}{
		{
			name: "Test 1: Instance was deleted",
			initialCache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
				plusInstance.GetInstanceMeta().GetInstanceId(): {
					InstanceId:           plusInstance.GetInstanceMeta().GetInstanceId(),
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
				},
			},
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
			initialCache: map[string]*mpi.InstanceHealth{
				ossInstance.GetInstanceMeta().GetInstanceId(): protos.HealthyInstanceHealth(),
			},
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
			healthWatcher.cache = test.initialCache
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
