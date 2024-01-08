package nginx

import (
	"fmt"
	"strings"

	"github.com/nginx/agent/v3/internal/models"
	"github.com/shirou/gopsutil/v3/process"
)

func GetInstances(processes []*process.Process) ([]*instances.Instance, error) {
	var processList []*instances.Instance

	nginxProcesses := make(map[int32]*process.Process)
	for _, p := range processes {

		name, _ := p.Name()
		cmd, _ := p.Cmdline()

		if isNginxProcess(name, cmd) {
			nginxProcesses[p.Pid] = p
		}
	}

	for pid, nginxProcess := range nginxProcesses {
		ppid, _ := nginxProcess.Ppid()

		_, ok := nginxProcesses[ppid]
		if ok {
			newProcess := &instances.Instance{
				InstanceId: fmt.Sprint(pid),
				Type:       instances.Type_NGINX,
			}
			processList = append(processList, newProcess)
		}
	}

	return processList, nil
}

func isNginxProcess(name string, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}
