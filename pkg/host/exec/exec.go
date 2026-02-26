// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/shirou/gopsutil/v4/host"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2 -generate
//counterfeiter:generate . ExecInterface
type ExecInterface interface {
	RunCmd(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error)
	Executable() (string, error)
	FindExecutable(name string) (string, error)
	ProcessID() int32
	KillProcess(pid int32) error
	Hostname() (string, error)
	HostID(ctx context.Context) (string, error)
	ReleaseInfo(ctx context.Context) (releaseInfo *v1.ReleaseInfo, err error)
}

type Exec struct{}

// RunCmd executes a command with the given arguments and returns its output.
// It combines stdout and stderr into a single buffer.
// If the command fails, the error is returned along with any output that was produced.
func (*Exec) RunCmd(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error) {
	command := exec.CommandContext(ctx, cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return bytes.NewBuffer(output), err
	}

	return bytes.NewBuffer(output), nil
}

// Executable returns the path to the current executable.
func (*Exec) Executable() (string, error) {
	return os.Executable()
}

// FindExecutable searches for an executable named by the given file name in the
// directories listed in the PATH environment variable.
func (*Exec) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

// ProcessID returns the process ID of the current process.
func (*Exec) ProcessID() int32 {
	return int32(os.Getpid())
}

// KillProcess sends a SIGHUP signal to the process with the given pid.
func (*Exec) KillProcess(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGHUP)
}

// Hostname returns the host name reported by the kernel.
func (*Exec) Hostname() (string, error) {
	return os.Hostname()
}

// HostID returns a unique ID for the host machine.
// The context can be used to cancel the operation.
func (*Exec) HostID(ctx context.Context) (string, error) {
	return host.HostIDWithContext(ctx)
}

// ReleaseInfo returns operating system release information.
// It provides details about the platform, version, and other system information.
func (*Exec) ReleaseInfo(ctx context.Context) (*v1.ReleaseInfo, error) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return &v1.ReleaseInfo{}, err
	}

	return &v1.ReleaseInfo{
		VersionId: hostInfo.PlatformVersion,
		Version:   hostInfo.KernelVersion,
		Codename:  hostInfo.OS,
		Name:      hostInfo.PlatformFamily,
		Id:        hostInfo.Platform,
	}, nil
}
