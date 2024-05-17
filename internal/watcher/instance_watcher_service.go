// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processWatcherOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processParser

type (
	processWatcherOperator interface {
		Processes(ctx context.Context) ([]*model.Process, error)
	}

	processParser interface {
		Parse(ctx context.Context, processes []*model.Process) []*v1.Instance
	}

	InstanceWatcherService struct {
		agentConfig    *config.Config
		processWatcher processWatcherOperator
		processParsers []processParser
		cache          []*v1.Instance
	}

	InstanceUpdates struct {
		newInstances     []*v1.Instance
		deletedInstances []*v1.Instance
	}
)

func NewInstanceWatcherService(agentConfig *config.Config) *InstanceWatcherService {
	return &InstanceWatcherService{
		agentConfig:    agentConfig,
		processWatcher: NewProcessWatcher(),
		processParsers: []processParser{
			NewNginxProcessParser(),
		},
		cache: []*v1.Instance{},
	}
}

func (iw *InstanceWatcherService) Watch(ctx context.Context, ch chan<- InstanceUpdates) {
	monitoringFrequency := iw.agentConfig.Watchers.InstanceWatcher.MonitoringFrequency
	slog.DebugContext(ctx, "Starting instance watcher monitoring", "monitoring_frequency", monitoringFrequency)

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-instanceWatcherTicker.C:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, logger.GenerateCorrelationID())
			instanceUpdates, err := iw.updates(newCtx)
			if err != nil {
				slog.ErrorContext(newCtx, "Instance watcher updates", "error", err)
			} else {
				ch <- instanceUpdates
			}
		}
	}
}

func (iw *InstanceWatcherService) updates(ctx context.Context) (
	instanceUpdates InstanceUpdates,
	err error,
) {
	processes, err := iw.processWatcher.Processes(ctx)
	if err != nil {
		return instanceUpdates, err
	}

	instancesFound := []*v1.Instance{}

	for _, processParser := range iw.processParsers {
		instancesFound = append(instancesFound, processParser.Parse(ctx, processes)...)
	}

	newInstances, deletedInstances := compareInstances(iw.cache, instancesFound)
	instanceUpdates.newInstances = newInstances
	instanceUpdates.deletedInstances = deletedInstances

	iw.cache = instancesFound

	return instanceUpdates, nil
}

func compareInstances(oldInstances, instances []*v1.Instance) (newInstances, deletedInstances []*v1.Instance) {
	instancesMap := make(map[int32]*v1.Instance)
	oldInstancesMap := make(map[int32]*v1.Instance)

	for _, instance := range instances {
		instancesMap[instance.GetInstanceRuntime().GetProcessId()] = instance
	}

	for _, oldInstance := range oldInstances {
		oldInstancesMap[oldInstance.GetInstanceRuntime().GetProcessId()] = oldInstance
	}

	for pid, instance := range instancesMap {
		_, ok := oldInstancesMap[pid]
		if !ok {
			newInstances = append(newInstances, instance)
		}
	}

	for pid, oldInstance := range oldInstancesMap {
		_, ok := instancesMap[pid]
		if !ok {
			deletedInstances = append(deletedInstances, oldInstance)
		}
	}

	return newInstances, deletedInstances
}
