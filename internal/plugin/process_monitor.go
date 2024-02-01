// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/datasource/host"
	"github.com/nginx/agent/v3/internal/model"
)

type GetProcessesFunc func() ([]*model.Process, error)

type ProcessMonitorParameters struct {
	MonitoringFrequency time.Duration
	getProcessesFunc    GetProcessesFunc
}

type ProcessMonitor struct {
	params      *ProcessMonitorParameters
	processes   []*model.Process
	messagePipe bus.MessagePipeInterface
}

func NewProcessMonitor(params *ProcessMonitorParameters) *ProcessMonitor {
	if params.getProcessesFunc == nil {
		params.getProcessesFunc = host.GetProcesses
	}

	return &ProcessMonitor{
		params:    params,
		processes: []*model.Process{},
	}
}

func (pm *ProcessMonitor) Init(messagePipe bus.MessagePipeInterface) {
	pm.messagePipe = messagePipe
	go pm.run(messagePipe.Context())
}

func (*ProcessMonitor) Close() {}

func (*ProcessMonitor) Info() *bus.Info {
	return &bus.Info{
		Name: "process-monitor",
	}
}

func (*ProcessMonitor) Process(*bus.Message) {}

func (*ProcessMonitor) Subscriptions() []string {
	return []string{}
}

func (pm *ProcessMonitor) run(ctx context.Context) {
	slog.Debug("Process monitor started", "monitoringPeriod", pm.params.MonitoringFrequency)

	processes, err := pm.params.getProcessesFunc()
	if err == nil {
		pm.processes = processes
		pm.messagePipe.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: processes})
	}

	slog.Debug("Processes updated")

	ticker := time.NewTicker(pm.params.MonitoringFrequency)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processes, err := pm.params.getProcessesFunc()
			if err != nil {
				slog.Error("Unable to get process information", "error", err)

				continue
			}

			pm.processes = processes
			pm.messagePipe.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: processes})
			slog.Debug("Processes updated")
		}
	}
}
