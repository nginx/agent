/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package exec

import (
	"bytes"
	"os/exec"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ExecInterface
//nolint:unused
type ExecInterface interface {
	RunCmd(cmd string, args ...string) (*bytes.Buffer, error)
	FindExecutable(name string) (string, error)
}

type Exec struct{}

func (*Exec) RunCmd(cmd string, args ...string) (*bytes.Buffer, error) {
	command := exec.Command(cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return bytes.NewBuffer(output), err
	}

	return bytes.NewBuffer(output), nil
}

func (*Exec) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}
