// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"
	"fmt"
	"strings"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/shirou/gopsutil/v3/process"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ProcessOperatorInterface
type (
	ProcessOperator struct{}

	ProcessOperatorInterface interface {
		Processes(ctx context.Context) ([]*model.Process, error)
		Process(ctx context.Context, PID int32) (*model.Process, error)
	}
)

var _ ProcessOperatorInterface = (*ProcessOperator)(nil)

func NewProcessOperator() *ProcessOperator {
	return &ProcessOperator{}
}

func (pw *ProcessOperator) Processes(ctx context.Context) ([]*model.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	internalProcesses := []*model.Process{}

	for _, proc := range processes {
		ppid, _ := proc.PpidWithContext(ctx)
		name, _ := proc.NameWithContext(ctx)
		cmd, _ := proc.CmdlineWithContext(ctx)
		exe, _ := proc.ExeWithContext(ctx)
		status, _ := proc.StatusWithContext(ctx)
		running, _ := proc.IsRunningWithContext(ctx)

		internalProcesses = append(internalProcesses, &model.Process{
			PID:     proc.Pid,
			PPID:    ppid,
			Name:    name,
			Cmd:     cmd,
			Exe:     exe,
			Status:  strings.Join(status, " "),
			Running: running,
		})
	}

	return internalProcesses, nil
}

func (pw *ProcessOperator) Process(ctx context.Context, pid int32) (*model.Process, error) {
	newProc, err := process.NewProcessWithContext(ctx, pid)
	status, _ := newProc.StatusWithContext(ctx)
	running, _ := newProc.IsRunningWithContext(ctx)
	ppid, _ := newProc.PpidWithContext(ctx)
	name, _ := newProc.NameWithContext(ctx)
	cmd, _ := newProc.CmdlineWithContext(ctx)
	exe, _ := newProc.ExeWithContext(ctx)

	if err != nil {
		return nil, fmt.Errorf("unable to create process with pid: %d, error : %w",
			pid, err)
	}

	proc := &model.Process{
		PID:     newProc.Pid,
		PPID:    ppid,
		Name:    name,
		Cmd:     cmd,
		Exe:     exe,
		Status:  strings.Join(status, " "),
		Running: running,
	}

	return proc, nil
}
