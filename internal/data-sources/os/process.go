package os

import (
	"github.com/nginx/agent/v3/internal/models/os"
	"github.com/shirou/gopsutil/v3/process"
)

func GetProcesses() ([]*os.Process, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	internalProcesses := []*os.Process{}

	for _, proc := range processes {
		ppid, _ := proc.Ppid()
		name, _ := proc.Name()
		cmd, _ := proc.Cmdline()

		internalProcesses = append(internalProcesses, &os.Process{
			Pid:  proc.Pid,
			Ppid: ppid,
			Name: name,
			Cmd:  cmd,
		})
	}

	return internalProcesses, nil
}
