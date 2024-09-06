// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package exec

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/shirou/gopsutil/v3/host"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ExecInterface
type ExecInterface interface {
	RunCmd(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error)
	Executable() (string, error)
	FindExecutable(name string) (string, error)
	ProcessID() int32
	KillProcess(pid int32) error
	Hostname() (string, error)
	HostID(ctx context.Context) (string, error)
	ReleaseInfo(ctx context.Context) (releaseInfo *v1.ReleaseInfo)
}

type Exec struct{}

func (*Exec) RunCmd(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error) {
	command := exec.CommandContext(ctx, cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return bytes.NewBuffer(output), err
	}

	return bytes.NewBuffer(output), nil
}

func (*Exec) Executable() (string, error) {
	return os.Executable()
}

func (*Exec) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

func (*Exec) ProcessID() int32 {
	return int32(os.Getpid())
}

func (*Exec) KillProcess(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGHUP)
}

func (*Exec) Hostname() (string, error) {
	return os.Hostname()
}

func (*Exec) HostID(ctx context.Context) (string, error) {
	return host.HostIDWithContext(ctx)
}

func (*Exec) ReleaseInfo(ctx context.Context) (releaseInfo *v1.ReleaseInfo) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not read release information for host", "error", err)
		return &v1.ReleaseInfo{}
	}

	return &v1.ReleaseInfo{
		VersionId: hostInfo.PlatformVersion,
		Version:   hostInfo.KernelVersion,
		Codename:  hostInfo.OS,
		Name:      hostInfo.PlatformFamily,
		Id:        hostInfo.Platform,
	}
}
