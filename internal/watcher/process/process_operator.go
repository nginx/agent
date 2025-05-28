// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
	"strings"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ProcessOperatorInterface
type (
	// ProcessOperator provides details about running NGINX processes.
	ProcessOperator struct{}

	ProcessOperatorInterface interface {
		Processes(ctx context.Context) (
			nginxProcesses []*nginxprocess.Process,
			nginxAppProtectProcesses []*nginxprocess.Process,
			err error,
		)
		Process(ctx context.Context, pid int32) (*nginxprocess.Process, error)
	}
)

func nginxFilter(ctx context.Context, p *process.Process) bool {
	name, _ := p.NameWithContext(ctx) // slow: shells out to ps
	if name != "nginx" {
		return false
	}

	cmdLine, _ := p.CmdlineWithContext(ctx) // slow: shells out to ps
	// ignore nginx processes in the middle of an upgrade
	if !strings.HasPrefix(cmdLine, "nginx:") || strings.Contains(cmdLine, "upgrade") {
		return false
	}

	return true
}

func napFilter(ctx context.Context, p *process.Process) bool {
	name, _ := p.NameWithContext(ctx) // slow: shells out to ps
	return name == "bd-socket-plugin"
}

var _ ProcessOperatorInterface = (*ProcessOperator)(nil)

func NewProcessOperator() *ProcessOperator {
	return &ProcessOperator{}
}

func (pw *ProcessOperator) Processes(ctx context.Context) (
	nginxProcesses []*nginxprocess.Process,
	nginxAppProtectProcesses []*nginxprocess.Process,
	err error,
) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	var filteredNginxProcesses []*process.Process

	for _, p := range processes {
		if nginxFilter(ctx, p) {
			filteredNginxProcesses = append(filteredNginxProcesses, p)
		} else if napFilter(ctx, p) {
			nginxAppProtectProcesses = append(nginxAppProtectProcesses, convertProcess(ctx, p))
		}
	}

	nginxProcesses, err = nginxprocess.ListWithProcesses(ctx, filteredNginxProcesses)
	if err != nil {
		return nil, nil, err
	}

	return nginxProcesses, nginxAppProtectProcesses, nil
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
