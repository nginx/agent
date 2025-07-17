// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package nginxprocess contains utilities for working with OS-level NGINX processes.
package nginxprocess

import (
	"context"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Process contains a snapshot of read-only data about an OS-level NGINX process. Create using [List] or [Find].
type Process struct {
	// Created is when this process was created, precision varies by platform and is at best to the millisecond. On
	// linux there can be significant skew compared to [time.Now], ± 1s.
	Created time.Time
	Name    string
	Cmd     string
	Exe     string // path to the executable
	Status  string // process status, only present if this process was created using [WithStatus]
	PID     int32
	PPID    int32 // parent PID
}

// IsWorker returns true if the process is a NGINX worker process.
func (p *Process) IsWorker() bool { return strings.HasPrefix(p.Cmd, "nginx: worker") }

// IsMaster returns true if the process is a NGINX master process.
func (p *Process) IsMaster() bool {
	return strings.HasPrefix(p.Cmd, "nginx: master") ||
		strings.HasPrefix(p.Cmd, "{nginx-debug} nginx: master")
}

// IsShuttingDown returns true if the process is shutting down. This can identify workers that are in the process of a
// graceful shutdown. See [changing NGINX configuration] for more details.
//
// [changing NGINX configuration]: https://nginx.org/en/docs/control.html#reconfiguration
func (p *Process) IsShuttingDown() bool { return strings.Contains(p.Cmd, "is shutting down") }

// IsHealthy uses Status flags to judge process health. Only works on processes created using [WithStatus].
func (p *Process) IsHealthy() bool {
	return p.Status != "" && !strings.Contains(p.Status, process.Zombie)
}

type options struct {
	loadStatus bool
}

// Option customizes how processes are gathered from the OS.
type Option interface{ apply(opts *options) }

type optionFunc func(*options)

//nolint:ireturn
func (f optionFunc) apply(o *options) { f(o) }

// WithStatus runs an additional lookup to load the process status.
func WithStatus(v bool) Option { //nolint:ireturn // functional options can be opaque
	return optionFunc(func(o *options) { o.loadStatus = v })
}

func convert(ctx context.Context, p *process.Process, o options) (*Process, error) {
	if err := ctx.Err(); err != nil { // fail fast if we've canceled
		return nil, err
	}

	name, _ := p.NameWithContext(ctx) // slow: shells out to ps
	if name != "nginx" && name != "nginx-debug" {
		return nil, errNotAnNginxProcess
	}

	cmdLine, _ := p.CmdlineWithContext(ctx) // slow: shells out to ps
	// ignore nginx processes in the middle of an upgrade

	if strings.Contains(cmdLine, "upgrade") {
		return nil, errNotAnNginxProcess
	}

	if strings.HasPrefix(cmdLine, "nginx:") || strings.HasPrefix(cmdLine, "{nginx-debug} nginx:") {
		var status string
		if o.loadStatus {
			flags, _ := p.StatusWithContext(ctx) // slow: shells out to ps
			status = strings.Join(flags, " ")
		}

		// unconditionally run fast lookups
		var created time.Time
		if millisSinceEpoch, err := p.CreateTimeWithContext(ctx); err == nil {
			created = time.UnixMilli(millisSinceEpoch)
		}
		ppid, _ := p.PpidWithContext(ctx)
		exe, _ := p.ExeWithContext(ctx)

		return &Process{
			PID:     p.Pid,
			PPID:    ppid,
			Name:    name,
			Cmd:     cmdLine,
			Created: created,
			Status:  status,
			Exe:     exe,
		}, ctx.Err()
	}

	return nil, errNotAnNginxProcess
}

// List returns a slice of all NGINX processes. Returns a zero-length slice if no NGINX processes are found.
func List(ctx context.Context, opts ...Option) (ret []*Process, err error) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return ListWithProcesses(ctx, processes, opts...)
}

// ListWithProcesses returns a slice of all NGINX processes.
// Returns a zero-length slice if no NGINX processes are found.
func ListWithProcesses(
	ctx context.Context,
	processes []*process.Process,
	opts ...Option,
) (ret []*Process, err error) {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}

	for _, p := range processes {
		pr, cerr := convert(ctx, p, o)
		if IsNotNginxErr(cerr) {
			continue
		}
		if cerr != nil {
			return nil, cerr
		}
		ret = append(ret, pr)
	}

	return ret, nil
}

// Find returns a single NGINX process by PID. Returns an error if the PID is no longer running or if it is not an NGINX
// process. Use with [IsProcessNotRunningErr] and [IsNotNginxErr].
func Find(ctx context.Context, pid int32, opts ...Option) (*Process, error) {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}
	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil, err
	}
	pr, err := convert(ctx, p, o)
	if err != nil {
		return nil, err
	}

	return pr, nil
}
