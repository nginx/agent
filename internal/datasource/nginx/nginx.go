/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nginx

import (
	"strings"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/datasource/nginx/process"
	"github.com/nginx/agent/v3/internal/datasource/os/exec"
	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/nginx/agent/v3/internal/uuid"
)

type GetInfo func(pid int32, exe string) (*process.Info, error)

type Nginx struct {
	parameters NginxParameters
}

type NginxParameters struct {
	GetInfo GetInfo
}

func New(parameters NginxParameters) *Nginx {
	if parameters.GetInfo == nil {
		parameters.GetInfo = process.New(&exec.Exec{}).GetInfo
	}
	return &Nginx{
		parameters: parameters,
	}
}

func (n *Nginx) GetInstances(processes []*os.Process) ([]*instances.Instance, error) {
	var processList []*instances.Instance

	nginxProcesses := make(map[int32]*os.Process)
	for _, p := range processes {
		if isNginxProcess(p.Name, p.Cmd) {
			nginxProcesses[p.Pid] = p
		}
	}

	for _, nginxProcess := range nginxProcesses {
		_, ok := nginxProcesses[nginxProcess.Ppid]
		if !ok {
			nginxInfo, err := n.parameters.GetInfo(nginxProcess.Pid, nginxProcess.Exe)

			nginxType := instances.Type_NGINX
			version := nginxInfo.Version

			if nginxInfo.PlusVersion != "" {
				nginxType = instances.Type_NGINXPLUS
				version = nginxInfo.PlusVersion
			}

			if err == nil {
				newProcess := &instances.Instance{
					InstanceId: uuid.Generate("%s_%s_%s", nginxProcess.Exe, nginxInfo.ConfPath, nginxInfo.Prefix),
					Type:       nginxType,
					Version:    version,
				}
				processList = append(processList, newProcess)
			}
		}
	}

	return processList, nil
}

func isNginxProcess(name string, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}
