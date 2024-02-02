// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bufio"
	"bytes"
	"log/slog"
	"path"
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	process "github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/uuid"
)

var (
	re     = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+)`)
	plusre = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+).\((?P<plus>\S+plus\S+)\)`)
)

type Info struct {
	Version       string
	PlusVersion   string
	Prefix        string
	ConfPath      string
	ConfigureArgs map[string]interface{}
	ExePath       string
}

type Nginx struct {
	executer exec.ExecInterface
}

type NginxParameters struct {
	executer exec.ExecInterface
}

func NewNginx(parameters NginxParameters) *Nginx {
	if parameters.executer == nil {
		parameters.executer = &exec.Exec{}
	}

	return &Nginx{
		executer: parameters.executer,
	}
}

// nolint: unparam // always returns nil but is a test
func (n *Nginx) GetInstances(processes []*model.Process) ([]*instances.Instance, error) {
	var processList []*instances.Instance

	nginxProcesses := make(map[int32]*model.Process)
	for _, p := range processes {
		if isNginxProcess(p.Name, p.Cmd) {
			nginxProcesses[p.Pid] = p
		}
	}

	for _, nginxProcess := range nginxProcesses {
		_, ok := nginxProcesses[nginxProcess.Ppid]
		if !ok {
			exe := nginxProcess.Exe
			if exe == "" {
				exe = process.New(n.executer).GetExe()
				if exe == "" {
					slog.Debug("Unable to find NGINX exe", "pid", nginxProcess.Pid)

					continue
				}
				nginxProcess.Exe = exe
			}

			nginxInfo, err := n.getInfo(exe)
			if err != nil {
				slog.Debug("Unable to get NGINX info", "pid", nginxProcess.Pid, "exe", exe)

				continue
			}

			processList = append(processList, convertInfoToProcess(*nginxInfo))
		}
	}

	return processList, nil
}

func (n *Nginx) getInfo(exePath string) (*Info, error) {
	var nginxInfo *Info

	outputBuffer, err := n.executer.RunCmd(exePath, "-V")
	if err != nil {
		return nil, err
	}

	nginxInfo = parseNginxVersionCommandOutput(outputBuffer)

	nginxInfo.ExePath = exePath

	return nginxInfo, err
}

func convertInfoToProcess(nginxInfo Info) *instances.Instance {
	nginxType := instances.Type_NGINX
	version := nginxInfo.Version

	if nginxInfo.PlusVersion != "" {
		nginxType = instances.Type_NGINX_PLUS
		version = nginxInfo.PlusVersion
	}

	return &instances.Instance{
		InstanceId: uuid.Generate("%s_%s_%s", nginxInfo.ExePath, nginxInfo.ConfPath, nginxInfo.Prefix),
		Type:       nginxType,
		Version:    version,
		Meta: &instances.Meta{
			Meta: &instances.Meta_NginxMeta{
				NginxMeta: &instances.NginxMeta{
					ConfigPath: nginxInfo.ConfPath,
					ExePath:    nginxInfo.ExePath,
				},
			},
		},
	}
}

func isNginxProcess(name, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}

func parseNginxVersionCommandOutput(output *bytes.Buffer) *Info {
	nginxInfo := &Info{}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "nginx version"):
			nginxInfo.Version, nginxInfo.PlusVersion = parseNginxVersion(line)
		case strings.HasPrefix(line, "configure arguments"):
			nginxInfo.ConfigureArgs = parseConfigureArguments(line)
		}
	}

	if nginxInfo.ConfigureArgs["prefix"] != nil {
		var ok bool
		nginxInfo.Prefix, ok = nginxInfo.ConfigureArgs["prefix"].(string)
		if !ok {
			slog.Error("failed to cast nginxInfo prefix to string")
			return nil
		}
	} else {
		nginxInfo.Prefix = "/usr/local/nginx"
	}

	if nginxInfo.ConfigureArgs["conf-path"] != nil {
		var ok bool
		nginxInfo.ConfPath, ok = nginxInfo.ConfigureArgs["conf-path"].(string)
		if !ok {
			slog.Error("failed to cast nginxInfo conf-path to string")
			return nil
		}
	} else {
		nginxInfo.ConfPath = path.Join(nginxInfo.Prefix, "/conf/nginx.conf")
	}

	return nginxInfo
}

func parseNginxVersion(line string) (version, plusVersion string) {
	matches := re.FindStringSubmatch(line)
	plusMatches := plusre.FindStringSubmatch(line)

	if len(plusMatches) > 0 {
		subNames := plusre.SubexpNames()
		for i, v := range plusMatches {
			switch subNames[i] {
			case "plus":
				plusVersion = v
			case "version":
				version = v
			}
		}

		return version, plusVersion
	}

	if len(matches) > 0 {
		for i, key := range re.SubexpNames() {
			val := matches[i]
			if key == "version" {
				version = val
			}
		}
	}

	return version, plusVersion
}

func parseConfigureArguments(line string) map[string]interface{} {
	// need to check for empty strings
	flags := strings.Split(line[len("configure arguments:"):], " --")
	result := make(map[string]interface{})

	for _, flag := range flags {
		vals := strings.Split(flag, "=")
		if isFlag(vals) {
			result[vals[0]] = true
		} else if isKeyValueFlag(vals) {
			result[vals[0]] = vals[1]
		}
	}

	return result
}

func isFlag(vals []string) bool {
	return len(vals) == 1 && vals[0] != ""
}

// nolint: gomnd
func isKeyValueFlag(vals []string) bool {
	return len(vals) == 2
}
