// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . healthWatcherOperator

type (
	healthWatcherOperator interface {
		Health(ctx context.Context, instanceID string) *v1.InstanceHealth
	}

	HealthWatcherService struct {
		agentConfig *config.Config
		cache       map[string]*v1.InstanceHealth    // key is instanceID
		watchers    map[string]healthWatcherOperator // key is instanceID
	}

	InstanceHealthMessage struct {
		correlationID  slog.Attr
		instanceHealth []*v1.InstanceHealth
	}
)

func NewHealthWatcherService(agentConfig *config.Config) *HealthWatcherService {
	return &HealthWatcherService{
		watchers:    make(map[string]healthWatcherOperator),
		cache:       make(map[string]*v1.InstanceHealth),
		agentConfig: agentConfig,
	}
}

func (hw *HealthWatcherService) AddHealthWatcher(instances []*v1.Instance) {
	for _, instance := range instances {
		switch instance.GetInstanceMeta().GetInstanceType() {
		case v1.InstanceMeta_INSTANCE_TYPE_NGINX, v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS:
			watcher := NewNginxHealthWatcher()
			hw.watchers[instance.GetInstanceMeta().GetInstanceId()] = watcher
		case v1.InstanceMeta_INSTANCE_TYPE_AGENT:
		case v1.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED,
			v1.InstanceMeta_INSTANCE_TYPE_UNIT:
			fallthrough
		default:
			slog.Warn("Health watcher not implemented", "instance_type",
				instance.GetInstanceMeta().GetInstanceType())
		}
	}
}

func (hw *HealthWatcherService) DeleteHealthWatcher(instances []*v1.Instance) {
	for _, instance := range instances {
		delete(hw.watchers, instance.GetInstanceMeta().GetInstanceId())
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
					correlationID:  correlationID,
					instanceHealth: healthStatuses,
				}
			} else {
				slog.DebugContext(newCtx, "Instance health watcher found no health updates")
			}
		}
	}
}

func (hw *HealthWatcherService) health(ctx context.Context) (healthStatuses []*v1.InstanceHealth, isHealthDiff bool,
) {
	currentHealth := make(map[string]*v1.InstanceHealth, len(hw.watchers))

	for instanceID, watcher := range hw.watchers {
		instanceHealth := watcher.Health(ctx, instanceID)
		healthStatuses = append(healthStatuses, instanceHealth)
		currentHealth[instanceID] = instanceHealth
	}

	isHealthDiff = hw.compareHealth(currentHealth)

	if isHealthDiff {
		hw.updateCache(currentHealth)
	}

	return healthStatuses, isHealthDiff
}

func (hw *HealthWatcherService) updateCache(currentHealth map[string]*v1.InstanceHealth) {
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

func (hw *HealthWatcherService) compareHealth(currentHealth map[string]*v1.InstanceHealth) bool {
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
