// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	process "github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/uuid"
)

var (
	ossVersionRegex  = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+)`)
	plusVersionRegex = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+).\((?P<plus>\S+plus\S+)\)`)
)

type Info struct {
	ProcessID       int32
	Version         string
	PlusVersion     string
	Prefix          string
	ConfPath        string
	ConfigureArgs   map[string]interface{}
	ExePath         string
	LoadableModules []string
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

func (n *Nginx) GetInstances(ctx context.Context, processes []*model.Process) []*v1.Instance {
	var processList []*v1.Instance

	nginxProcesses := n.getNginxProcesses(processes)

	for _, nginxProcess := range nginxProcesses {
		_, ok := nginxProcesses[nginxProcess.Ppid]
		if !ok {
			nginxInfo, err := n.getInfo(ctx, nginxProcess)
			if err != nil {
				slog.DebugContext(ctx, "Unable to get NGINX info", "pid", nginxProcess.Pid, "error", err)

				continue
			}

			processList = append(processList, convertInfoToProcess(*nginxInfo))
		}
	}

	return processList
}

func (*Nginx) getNginxProcesses(processes []*model.Process) map[int32]*model.Process {
	nginxProcesses := make(map[int32]*model.Process)

	for _, p := range processes {
		if isNginxProcess(p.Name, p.Cmd) {
			nginxProcesses[p.Pid] = p
		}
	}

	return nginxProcesses
}

func (n *Nginx) getInfo(ctx context.Context, nginxProcess *model.Process) (*Info, error) {
	exePath := nginxProcess.Exe

	if exePath == "" {
		exePath = process.New(n.executer).GetExe(ctx)
		if exePath == "" {
			return nil, fmt.Errorf("unable to find NGINX exe for process %d", nginxProcess.Pid)
		}
	}

	var nginxInfo *Info

	outputBuffer, err := n.executer.RunCmd(ctx, exePath, "-V")
	if err != nil {
		return nil, err
	}

	nginxInfo = parseNginxVersionCommandOutput(ctx, outputBuffer)

	nginxInfo.ExePath = exePath
	nginxInfo.ProcessID = nginxProcess.Pid

	loadableModules := getLoadableModules(nginxInfo)
	nginxInfo.LoadableModules = loadableModules
	slog.Info("", "", loadableModules)
	slog.Info("nginxInfo.LoadableModules", "", nginxInfo.LoadableModules)

	return nginxInfo, err
}

func convertInfoToProcess(nginxInfo Info) *v1.Instance {
	var instanceRuntime *v1.InstanceRuntime
	nginxType := v1.InstanceMeta_INSTANCE_TYPE_NGINX
	version := nginxInfo.Version

	if nginxInfo.PlusVersion == "" {
		instanceRuntime = &v1.InstanceRuntime{
			ProcessId:  nginxInfo.ProcessID,
			BinaryPath: nginxInfo.ExePath,
			ConfigPath: nginxInfo.ConfPath,
			Details: &v1.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &v1.NGINXRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: nginxInfo.LoadableModules,
					DynamicModules:  []string{},
				},
			},
		}
	} else {
		instanceRuntime = &v1.InstanceRuntime{
			ProcessId:  nginxInfo.ProcessID,
			BinaryPath: nginxInfo.ExePath,
			ConfigPath: nginxInfo.ConfPath,
			Details: &v1.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &v1.NGINXPlusRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: nginxInfo.LoadableModules,
					DynamicModules:  []string{},
					PlusApi:         "",
				},
			},
		}

		nginxType = v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS
		version = nginxInfo.PlusVersion
	}

	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   uuid.Generate("%s_%s_%s", nginxInfo.ExePath, nginxInfo.ConfPath, nginxInfo.Prefix),
			InstanceType: nginxType,
			Version:      version,
		},
		InstanceRuntime: instanceRuntime,
	}
}

func isNginxProcess(name, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}

func parseNginxVersionCommandOutput(ctx context.Context, output *bytes.Buffer) *Info {
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

	nginxInfo.Prefix = getNginxPrefix(ctx, nginxInfo)
	nginxInfo.ConfPath = getNginxConfPath(ctx, nginxInfo)

	return nginxInfo
}

func parseNginxVersion(line string) (version, plusVersion string) {
	ossVersionMatches := ossVersionRegex.FindStringSubmatch(line)
	plusVersionMatches := plusVersionRegex.FindStringSubmatch(line)

	ossSubNames := ossVersionRegex.SubexpNames()
	plusSubNames := plusVersionRegex.SubexpNames()

	for index, value := range ossVersionMatches {
		if ossSubNames[index] == "version" {
			version = value
		}
	}

	for index, value := range plusVersionMatches {
		switch plusSubNames[index] {
		case "plus":
			plusVersion = value
		case "version":
			version = value
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

func getNginxPrefix(ctx context.Context, nginxInfo *Info) string {
	var prefix string

	if nginxInfo.ConfigureArgs["prefix"] != nil {
		var ok bool
		prefix, ok = nginxInfo.ConfigureArgs["prefix"].(string)
		if !ok {
			slog.WarnContext(ctx, "Failed to cast nginxInfo prefix to string")
		}
	} else {
		prefix = "/usr/local/nginx"
	}

	return prefix
}

func getNginxConfPath(ctx context.Context, nginxInfo *Info) string {
	var confPath string

	if nginxInfo.ConfigureArgs["conf-path"] != nil {
		var ok bool
		confPath, ok = nginxInfo.ConfigureArgs["conf-path"].(string)
		if !ok {
			slog.WarnContext(ctx, "failed to cast nginxInfo conf-path to string")
		}
	} else {
		confPath = path.Join(nginxInfo.Prefix, "/conf/nginx.conf")
	}

	return confPath
}

func isFlag(vals []string) bool {
	return len(vals) == 1 && vals[0] != ""
}

// nolint: gomnd
func isKeyValueFlag(vals []string) bool {
	return len(vals) == 2
}

func getLoadableModules(nginxInfo *Info) (modules []string) {
	var err error
	if nginxInfo.ConfigureArgs["modules-path"] != nil {
		modulePath, ok := nginxInfo.ConfigureArgs["modules-path"].(string)
		if !ok {
			slog.Warn("error parsing modules-path")
			return modules
		}
		modules, err = readDirectory(modulePath, ".so")
		if err != nil {
			slog.Warn("error reading module dir", "dir", modulePath, "error", err)
			return modules
		}

		return modules
	}

	return modules
}

func readDirectory(dir, ext string) (files []string, err error) {
	dirInfo, err := os.ReadDir(dir)
	if err != nil {
		return files, fmt.Errorf("unable to read directory %s, %w", dir, err)
	}

	for _, file := range dirInfo {
		files = append(files, strings.ReplaceAll(file.Name(), ext, ""))
	}

	return files, err
}
