// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// nolint
type (
	healthWatcherOperator interface {
		Health(ctx context.Context, instanceID string) *v1.InstanceHealth
	}

	HealthWatcherService struct {
		agentConfig *config.Config
		cache       map[string]*v1.InstanceHealth // key is instanceID
		watchers    map[string]healthWatcherOperator
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
		case v1.InstanceMeta_INSTANCE_TYPE_AGENT,
			v1.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED,
			v1.InstanceMeta_INSTANCE_TYPE_UNIT:
			fallthrough
		default:
			slog.Warn("Not Implemented")
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
			var healthStatuses []*v1.InstanceHealth
			correlationID := logger.GenerateCorrelationID()
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, correlationID)

			for instanceID, watcher := range hw.watchers {
				health := watcher.Health(ctx, instanceID)
				healthStatuses = append(healthStatuses, health)
			}

			if len(healthStatuses) > 0 {
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
