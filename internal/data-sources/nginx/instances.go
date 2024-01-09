package nginx

import (
	"fmt"
	"strings"

	"github.com/nginx/agent/v3/internal/models/instances"
	"github.com/nginx/agent/v3/internal/models/os"
)

func GetInstances(processes []*os.Process) ([]*instances.Instance, error) {
	var processList []*instances.Instance

	nginxProcesses := make(map[int32]*os.Process)
	for _, p := range processes {
		if isNginxProcess(p.Name, p.Cmd) {
			nginxProcesses[p.Pid] = p
		}
	}

	for pid, nginxProcess := range nginxProcesses {
		_, ok := nginxProcesses[nginxProcess.Ppid]
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
