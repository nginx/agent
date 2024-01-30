/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nginx

import (
	"strings"

	"github.com/nginx/agent/v3/internal/datasource/os/exec"
)

type Process struct {
	executer exec.ExecInterface
}

func New(executer exec.ExecInterface) *Process {
	return &Process{executer: executer}
}

func (np *Process) GetExe() string {
	exe := ""

	out, commandErr := np.executer.RunCmd("sh", "-c", "command -v nginx")
	if commandErr == nil {
		exe = strings.TrimSuffix(out.String(), "\n")
	}

	if exe == "" {
		exe = np.defaultToNginxCommandForProcessPath()
	}

	if strings.Contains(exe, "(deleted)") {
		exe = np.sanitizeExeDeletedPath(exe)
	}

	return exe
}

func (np *Process) defaultToNginxCommandForProcessPath() string {
	path, err := np.executer.FindExecutable("nginx")
	if err != nil {
		return ""
	}
	return path
}

func (np *Process) sanitizeExeDeletedPath(exe string) string {
	firstSpace := strings.Index(exe, "(deleted)")
	if firstSpace != -1 {
		return strings.TrimSpace(exe[0:firstSpace])
	}
	return strings.TrimSpace(exe)
}
