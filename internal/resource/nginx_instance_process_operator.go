// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"errors"
	"log/slog"

	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/pkg/id"

	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
)

type NginxInstanceProcessOperator struct{}

var _ processOperator = (*NginxInstanceProcessOperator)(nil)

func NewNginxInstanceProcessOperator() *NginxInstanceProcessOperator {
	return &NginxInstanceProcessOperator{}
}

func (p *NginxInstanceProcessOperator) FindNginxProcesses(ctx context.Context) ([]*nginxprocess.Process, error) {
	processes, procErr := process.ProcessesWithContext(ctx)
	if procErr != nil {
		return nil, procErr
	}

	nginxProcesses, err := nginxprocess.ListWithProcesses(ctx, processes)
	if err != nil {
		return nil, err
	}

	return nginxProcesses, nil
}

func (p *NginxInstanceProcessOperator) NginxWorkerProcesses(ctx context.Context,
	masterProcessPid int32,
) []*nginxprocess.Process {
	slog.DebugContext(ctx, "Getting NGINX worker processes for NGINX reload")
	var workers []*nginxprocess.Process
	nginxProcesses, err := p.FindNginxProcesses(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get NGINX processes", "error", err)
		return workers
	}

	for _, nginxProcess := range nginxProcesses {
		if nginxProcess.IsWorker() && nginxProcess.PPID == masterProcessPid {
			workers = append(workers, nginxProcess)
		}
	}

	return workers
}

func (p *NginxInstanceProcessOperator) FindParentProcessID(ctx context.Context, instanceID string,
	nginxProcesses []*nginxprocess.Process, executer exec.ExecInterface,
) (int32, error) {
	var pid int32

	for _, proc := range nginxProcesses {
		if proc.IsMaster() {
			info, infoErr := nginx.ProcessInfo(ctx, proc, executer)
			if infoErr != nil {
				slog.WarnContext(ctx, "Failed to get NGINX process info from master process", "error", infoErr)
				continue
			}
			processInstanceID := id.Generate("%s_%s_%s", info.ExePath, info.ConfPath, info.Prefix)
			if instanceID == processInstanceID {
				slog.DebugContext(ctx, "Found NGINX process ID", "process_id", processInstanceID)
				return proc.PID, nil
			}
		}
	}

	return pid, errors.New("unable to find parent process")
}
