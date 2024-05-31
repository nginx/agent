// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . healthWatcherOperator

type (
	healthWatcherOperator interface {
		Health(ctx context.Context, instance *mpi.Instance) (*mpi.InstanceHealth, error)
	}

	HealthWatcherService struct {
		agentConfig *config.Config
		cache       map[string]*mpi.InstanceHealth   // key is instanceID
		watchers    map[string]healthWatcherOperator // key is instanceID
		instances   map[string]*mpi.Instance         // key is instanceID
	}

	InstanceHealthMessage struct {
		CorrelationID  slog.Attr
		InstanceHealth []*mpi.InstanceHealth
	}
)

func NewHealthWatcherService(agentConfig *config.Config) *HealthWatcherService {
	return &HealthWatcherService{
		watchers:    make(map[string]healthWatcherOperator),
		cache:       make(map[string]*mpi.InstanceHealth),
		instances:   make(map[string]*mpi.Instance),
		agentConfig: agentConfig,
	}
}

func (hw *HealthWatcherService) AddHealthWatcher(instances []*mpi.Instance) {
	for _, instance := range instances {
		switch instance.GetInstanceMeta().GetInstanceType() {
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX, mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS:
			watcher := NewNginxHealthWatcher()
			hw.watchers[instance.GetInstanceMeta().GetInstanceId()] = watcher
		case mpi.InstanceMeta_INSTANCE_TYPE_AGENT:
		case mpi.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED,
			mpi.InstanceMeta_INSTANCE_TYPE_UNIT:
			fallthrough
		default:
			slog.Warn("Health watcher not implemented", "instance_type",
				instance.GetInstanceMeta().GetInstanceType())
		}
		hw.instances[instance.GetInstanceMeta().GetInstanceId()] = instance
	}
}

func (hw *HealthWatcherService) UpdateHealthWatcher(instances []*mpi.Instance) {
	for _, instance := range instances {
		hw.instances[instance.GetInstanceMeta().GetInstanceId()] = instance
	}
}

func (hw *HealthWatcherService) DeleteHealthWatcher(instances []*mpi.Instance) {
	for _, instance := range instances {
		delete(hw.watchers, instance.GetInstanceMeta().GetInstanceId())
		delete(hw.instances, instance.GetInstanceMeta().GetInstanceId())
	}
}

func (hw *HealthWatcherService) Watch(ctx context.Context, ch chan<- InstanceHealthMessage) {
	monitoringFrequency := hw.agentConfig.Watchers.InstanceHealthWatcher.MonitoringFrequency
	slog.DebugContext(ctx, "Starting health watcher monitoring", "monitoring_frequency", monitoringFrequency)

	instanceHealthTicker := time.NewTicker(monitoringFrequency)
	defer instanceHealthTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-instanceHealthTicker.C:
			correlationID := logger.GenerateCorrelationID()
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, correlationID)

			healthStatuses, isHealthDiff := hw.health(ctx)
			if isHealthDiff && len(healthStatuses) > 0 {
				ch <- InstanceHealthMessage{
					CorrelationID:  correlationID,
					InstanceHealth: healthStatuses,
				}
			} else {
				slog.DebugContext(newCtx, "Instance health watcher found no health updates")
			}
		}
	}
}

func (hw *HealthWatcherService) health(ctx context.Context) (updatedStatuses []*mpi.InstanceHealth, isHealthDiff bool,
) {
	currentHealth := make(map[string]*mpi.InstanceHealth, len(hw.watchers))
	allStatuses := make([]*mpi.InstanceHealth, 0)

	for instanceID, watcher := range hw.watchers {
		instanceHealth, err := watcher.Health(ctx, hw.instances[instanceID])
		if instanceHealth == nil {
			instanceHealth = &mpi.InstanceHealth{
				InstanceId:           instanceID,
				InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNSPECIFIED,
				Description: fmt.Sprintf("failed to get health for instance %s, error: %v",
					instanceID, err.Error()),
			}
		}
		allStatuses = append(allStatuses, instanceHealth)
		currentHealth[instanceID] = instanceHealth
	}

	isHealthDiff = hw.compareHealth(currentHealth)

	if isHealthDiff {
		hw.updateCache(currentHealth)
	}

	updatedStatuses = hw.compareCache(allStatuses)

	return updatedStatuses, isHealthDiff
}

// update the cache with the most recent instance healths
func (hw *HealthWatcherService) updateCache(currentHealth map[string]*mpi.InstanceHealth) {
	for instanceID, healthStatus := range currentHealth {
		hw.cache[instanceID] = healthStatus
	}

	for key := range hw.cache {
		if _, ok := currentHealth[key]; !ok {
			delete(hw.cache, key)
		}
	}

	slog.Debug("Updating health watcher cache", "cache", hw.cache)
}

// compare the cache with the current list of instances to check if an instance has been deleted
// if an instance has been deleted add an UNHEALTHY health status to the list of instances for that instance
func (hw *HealthWatcherService) compareCache(healthStatuses []*mpi.InstanceHealth) []*mpi.InstanceHealth {
	if len(hw.cache) != len(hw.instances) {
		for instanceID := range hw.cache {
			if _, ok := hw.instances[instanceID]; !ok {
				health := &mpi.InstanceHealth{
					InstanceId:           instanceID,
					InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
					Description:          fmt.Sprintf("instance %s not found", instanceID),
				}
				healthStatuses = append(healthStatuses, health)
				delete(hw.cache, instanceID)
			}
		}
	}

	return healthStatuses
}

// compare current health with cached health to see if the health of an instance has changed
func (hw *HealthWatcherService) compareHealth(currentHealth map[string]*mpi.InstanceHealth) bool {
	if len(currentHealth) != len(hw.cache) {
		return true
	}

	for key, health := range currentHealth {
		if !proto.Equal(health, hw.cache[key]) {
			return true
		}
	}

	return false
}
