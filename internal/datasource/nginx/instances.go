/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nginx

import (
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/internal/model/instances"
	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/nginx/agent/v3/internal/util"
)

var (
	re     = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+)`)
	plusre = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+).\((?P<plus>\S+plus\S+)\)`)
)

type GetInfo func(pid int32, exe string) (*NginxInfo, error)

type Nginx struct {
	parameters NginxParameters
}

type NginxParameters struct {
	GetInfo GetInfo
}

func NewNginx(parameters NginxParameters) *Nginx {
	if parameters.GetInfo == nil {
		parameters.GetInfo = NewNginxProcess(&util.Helper{}).GetNginxInfo
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
			nginxInfo, err := n.parameters.GetInfo(nginxProcess.GetPid(), nginxProcess.GetExe())

			nginxType := instances.Type_NGINX
			version := nginxInfo.Version

			if nginxInfo.PlusVersion != "" {
				nginxType = instances.Type_NGINXPLUS
				version = nginxInfo.PlusVersion
			}

			if err == nil {
				newProcess := &instances.Instance{
					InstanceId: util.GenerateUUID("%s_%s_%s", nginxProcess.GetExe(), nginxInfo.ConfPath, nginxInfo.Prefix),
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
