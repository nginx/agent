// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package exec

import (
	"bytes"
	"context"
	"os/exec"
	"syscall"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ExecInterface
type ExecInterface interface {
	RunCmd(ctx context.Context, cmd string, args ...string) (*bytes.Buffer, error)
	FindExecutable(name string) (string, error)
	KillProcess(pid int32) error
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

func (*Exec) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

func (*Exec) KillProcess(pid int32) error {
	return syscall.Kill(int(pid), syscall.SIGHUP)
}
