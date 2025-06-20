// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"
	"strings"

	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ProcessOperatorInterface
type (
	// ProcessOperator provides details about running NGINX processes.
	ProcessOperator struct{}

	ProcessOperatorInterface interface {
		Processes(ctx context.Context) (
			nginxProcesses []*nginxprocess.Process,
			err error,
		)
		Process(ctx context.Context, pid int32) (*nginxprocess.Process, error)
	}
)

var _ ProcessOperatorInterface = (*ProcessOperator)(nil)

func NewProcessOperator() *ProcessOperator {
	return &ProcessOperator{}
}

func (pw *ProcessOperator) Processes(ctx context.Context) (
	nginxProcesses []*nginxprocess.Process,
	err error,
) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return nginxprocess.ListWithProcesses(ctx, processes)
}

func (pw *ProcessOperator) Process(ctx context.Context, pid int32) (*nginxprocess.Process, error) {
	proc, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil, err
	}

	return convertProcess(ctx, proc), nil
}

func convertProcess(ctx context.Context, proc *process.Process) *nginxprocess.Process {
	ppid, _ := proc.PpidWithContext(ctx)
	name, _ := proc.NameWithContext(ctx)
	cmd, _ := proc.CmdlineWithContext(ctx)
	exe, _ := proc.ExeWithContext(ctx)
	status, _ := proc.StatusWithContext(ctx)

	return &nginxprocess.Process{
		PID:    proc.Pid,
		PPID:   ppid,
		Name:   name,
		Cmd:    cmd,
		Exe:    exe,
		Status: strings.Join(status, " "),
	}
}
