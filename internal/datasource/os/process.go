/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package os

import (
	"github.com/nginx/agent/v3/internal/model"
	"github.com/shirou/gopsutil/v3/process"
)

func GetProcesses() ([]*model.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	internalProcesses := []*model.Process{}

	for _, proc := range processes {
		ppid, _ := proc.Ppid()
		name, _ := proc.Name()
		cmd, _ := proc.Cmdline()
		exe, _ := proc.Exe()

		internalProcesses = append(internalProcesses, &model.Process{
			Pid:  proc.Pid,
			Ppid: ppid,
			Name: name,
			Cmd:  cmd,
			Exe:  exe,
		})
	}

	return internalProcesses, nil
}
