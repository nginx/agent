/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package util

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os/exec"

	"github.com/google/uuid"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o mock_helper.go . HelperInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/util mock_helper.go | sed -e s\\/util\\\\.\\/\\/g > mock_helper_fixed.go"
//go:generate mv mock_helper_fixed.go mock_helper.go
type HelperInterface interface {
	RunCmd(cmd string, args ...string) (*bytes.Buffer, error)
	FindExecutable(name string) (string, error)
}

type Helper struct{}

func GenerateUUID(format string, a ...interface{}) string {
	h := sha256.New()
	s := fmt.Sprintf(format, a...)
	_, _ = h.Write([]byte(s))
	id := fmt.Sprintf("%x", h.Sum(nil))
	return uuid.NewMD5(uuid.Nil, []byte(id)).String()
}

func (*Helper) RunCmd(cmd string, args ...string) (*bytes.Buffer, error) {
	command := exec.Command(cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return bytes.NewBuffer(output), err
	}

	return bytes.NewBuffer(output), nil
}

func (*Helper) FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}
