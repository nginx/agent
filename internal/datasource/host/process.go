// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/shirou/gopsutil/v3/process"
)

func GetProcesses(ctx context.Context) ([]*model.Process, error) {
	slog.DebugContext(ctx, "Getting host processes")

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
