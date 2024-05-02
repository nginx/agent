// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host"
	"github.com/nginx/agent/v3/internal/logger"
)

type GetProcessesFunc func(ctx context.Context) (host.NginxProcesses, error)

type ProcessMonitor struct {
	monitoringFrequency time.Duration
	processes           host.NginxProcesses
	messagePipe         bus.MessagePipeInterface
	getProcessesFunc    GetProcessesFunc
	processTicker       *time.Ticker
	cancel              context.CancelFunc
	processesMutex      sync.Mutex
}

func NewProcessMonitor(agentConfig *config.Config) *ProcessMonitor {
	return &ProcessMonitor{
		monitoringFrequency: agentConfig.ProcessMonitor.MonitoringFrequency,
		processes:           make(host.NginxProcesses),
		getProcessesFunc:    host.GetNginxProcesses,
		processTicker:       nil,
		processesMutex:      sync.Mutex{},
	}
}

func (pm *ProcessMonitor) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting process monitor plugin", "monitoring_period", pm.monitoringFrequency)

	pm.messagePipe = messagePipe
	var pmCtx context.Context
	pmCtx, pm.cancel = context.WithCancel(ctx)
	go pm.run(pmCtx)

	return nil
}

func (pm *ProcessMonitor) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing process monitor plugin")

	pm.processesMutex.Lock()
	defer pm.processesMutex.Unlock()

	pm.processes = nil
	pm.cancel()

	return nil
}

func (*ProcessMonitor) Info() *bus.Info {
	return &bus.Info{
		Name: "process-monitor",
	}
}

func (*ProcessMonitor) Process(_ context.Context, _ *bus.Message) {}

func (*ProcessMonitor) Subscriptions() []string {
	return []string{}
}

func (pm *ProcessMonitor) getProcesses() host.NginxProcesses {
	pm.processesMutex.Lock()
	defer pm.processesMutex.Unlock()

	return pm.processes
}

func (pm *ProcessMonitor) run(ctx context.Context) {
	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, logger.GenerateCorrelationID())
	slog.DebugContext(newCtx, "Getting processes on startup")

	processes, err := pm.getProcessesFunc(newCtx)
	if err == nil {
		pm.processesMutex.Lock()
		pm.processes = processes
		pm.processesMutex.Unlock()

		pm.messagePipe.Process(newCtx, &bus.Message{Topic: bus.OsProcessesTopic, Data: processes})
	}

	slog.DebugContext(newCtx, "Processes updated")

	pm.processTicker = time.NewTicker(pm.monitoringFrequency)
	defer pm.processTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.processTicker.C:
			slog.DebugContext(ctx, "Checking for process updates")

			processes, err = pm.getProcessesFunc(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to get process information", "error", err)

				continue
			}

			pm.processesMutex.Lock()
			if haveProcessesChanged(pm.processes, processes) {
				procChangedCtx := context.WithValue(
					newCtx,
					logger.CorrelationIDContextKey,
					logger.GenerateCorrelationID(),
				)
				slog.DebugContext(procChangedCtx, "Processes changes detected")
				pm.processes = processes
				pm.messagePipe.Process(procChangedCtx, &bus.Message{Topic: bus.OsProcessesTopic, Data: processes})
			}
			pm.processesMutex.Unlock()
		}
	}
}

func haveProcessesChanged(oldProcesses, newProcesses host.NginxProcesses) bool {
	// Check if the number of processes has changed
	if len(oldProcesses) != len(newProcesses) {
		return true
	}

	processIDMap := make(map[int32]struct{})
	for _, oldProcess := range oldProcesses {
		processIDMap[oldProcess.Pid] = struct{}{}
	}

	// Check if the process IDs have changed
	for _, newProcess := range newProcesses {
		if _, ok := processIDMap[newProcess.Pid]; !ok {
			return true
		}
	}

	return false
}
