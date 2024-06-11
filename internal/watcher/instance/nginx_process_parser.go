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

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/uuid"
)

const (
	withWithPrefix   = "with-"
	withModuleSuffix = "module"
)

type (
	NginxProcessParser struct {
		executer exec.ExecInterface
	}

	Info struct {
		ProcessID       int32
		Version         string
		Prefix          string
		ConfPath        string
		ConfigureArgs   map[string]interface{}
		ExePath         string
		LoadableModules []string
		DynamicModules  []string
	}
)

var (
	_ processParser = (*NginxProcessParser)(nil)

	versionRegex = regexp.MustCompile(`(?P<name>\S+)\/(?P<version>.*)`)
)

func NewNginxProcessParser() *NginxProcessParser {
	return &NginxProcessParser{
		executer: &exec.Exec{},
	}
}

// cognitive complexity of 16 because of the if statements in the for loop
// don't think can be avoided due to the need for continue
// nolint: revive
func (npp *NginxProcessParser) Parse(ctx context.Context, processes []*model.Process) map[string]*mpi.Instance {
	instanceMap := make(map[string]*mpi.Instance)   // key is instanceID
	workers := make(map[int32][]*mpi.InstanceChild) // key is ppid of process

	nginxProcesses := filterNginxProcesses(processes)

	for _, nginxProcess := range nginxProcesses {
		// Here we are determining if the nginxProcess is a worker process
		if strings.Contains(nginxProcess.Cmd, "worker") {
			// Here we are determining if the worker process has a master
			if masterProcess, ok := nginxProcesses[nginxProcess.PPID]; ok {
				workers[masterProcess.PID] = append(workers[masterProcess.PID],
					&mpi.InstanceChild{ProcessId: nginxProcess.PID})

				continue
			}
			nginxInfo, err := npp.getInfo(ctx, nginxProcess)
			if err != nil {
				slog.DebugContext(ctx, "Unable to get NGINX info", "pid", nginxProcess.PID, "error", err)

				continue
			}
			// set instance process ID to 0 as there is no master process
			nginxInfo.ProcessID = 0

			instance := convertInfoToInstance(*nginxInfo)

			if foundInstance, ok := instanceMap[instance.GetInstanceMeta().GetInstanceId()]; ok {
				foundInstance.GetInstanceRuntime().InstanceChildren = append(foundInstance.GetInstanceRuntime().
					GetInstanceChildren(), &mpi.InstanceChild{ProcessId: nginxProcess.PID})

				continue
			}

			instance.GetInstanceRuntime().InstanceChildren = append(instance.GetInstanceRuntime().
				GetInstanceChildren(), &mpi.InstanceChild{ProcessId: nginxProcess.PID})

			instanceMap[instance.GetInstanceMeta().GetInstanceId()] = instance

			continue
		}

		// Here we are determining if the nginxProcess is a master process
		nginxInfo, err := npp.getInfo(ctx, nginxProcess)
		if err != nil {
			slog.DebugContext(ctx, "Unable to get NGINX info", "pid", nginxProcess.PID, "error", err)

			continue
		}

		instance := convertInfoToInstance(*nginxInfo)
		instanceMap[instance.GetInstanceMeta().GetInstanceId()] = instance
	}

	for _, instance := range instanceMap {
		if val, ok := workers[instance.GetInstanceRuntime().GetProcessId()]; ok {
			instance.InstanceRuntime.InstanceChildren = val
		}
	}

	return instanceMap
}

func (npp *NginxProcessParser) getInfo(ctx context.Context, nginxProcess *model.Process) (*Info, error) {
	exePath := nginxProcess.Exe

	if exePath == "" {
		exePath = npp.getExe(ctx)
		if exePath == "" {
			return nil, fmt.Errorf("unable to find NGINX exe for process %d", nginxProcess.PID)
		}
	}

	var nginxInfo *Info

	outputBuffer, err := npp.executer.RunCmd(ctx, exePath, "-V")
	if err != nil {
		return nil, err
	}

	nginxInfo = parseNginxVersionCommandOutput(ctx, outputBuffer)

	nginxInfo.ExePath = exePath
	nginxInfo.ProcessID = nginxProcess.PID

	loadableModules := getLoadableModules(nginxInfo)
	nginxInfo.LoadableModules = loadableModules

	nginxInfo.DynamicModules = getDynamicModules(nginxInfo)

	return nginxInfo, err
}

func (npp *NginxProcessParser) getExe(ctx context.Context) string {
	exePath := ""

	out, commandErr := npp.executer.RunCmd(ctx, "sh", "-c", "command -v nginx")
	if commandErr == nil {
		exePath = strings.TrimSuffix(out.String(), "\n")
	}

	if exePath == "" {
		exePath = npp.defaultToNginxCommandForProcessPath()
	}

	if strings.Contains(exePath, "(deleted)") {
		exePath = sanitizeExeDeletedPath(exePath)
	}

	return exePath
}

func (npp *NginxProcessParser) defaultToNginxCommandForProcessPath() string {
	exePath, err := npp.executer.FindExecutable("nginx")
	if err != nil {
		return ""
	}

	return exePath
}

func sanitizeExeDeletedPath(exe string) string {
	firstSpace := strings.Index(exe, "(deleted)")
	if firstSpace != -1 {
		return strings.TrimSpace(exe[0:firstSpace])
	}

	return strings.TrimSpace(exe)
}

func convertInfoToInstance(nginxInfo Info) *mpi.Instance {
	var instanceRuntime *mpi.InstanceRuntime
	nginxType := mpi.InstanceMeta_INSTANCE_TYPE_NGINX
	version := nginxInfo.Version

	if !strings.Contains(nginxInfo.Version, "plus") {
		instanceRuntime = &mpi.InstanceRuntime{
			ProcessId:  nginxInfo.ProcessID,
			BinaryPath: nginxInfo.ExePath,
			ConfigPath: nginxInfo.ConfPath,
			Details: &mpi.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &mpi.NGINXRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: nginxInfo.LoadableModules,
					DynamicModules:  nginxInfo.DynamicModules,
				},
			},
		}
	} else {
		instanceRuntime = &mpi.InstanceRuntime{
			ProcessId:  nginxInfo.ProcessID,
			BinaryPath: nginxInfo.ExePath,
			ConfigPath: nginxInfo.ConfPath,
			Details: &mpi.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &mpi.NGINXPlusRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: nginxInfo.LoadableModules,
					DynamicModules:  nginxInfo.DynamicModules,
					PlusApi:         "",
				},
			},
		}

		nginxType = mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS
		version = nginxInfo.Version
	}

	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   uuid.Generate("%s_%s_%s", nginxInfo.ExePath, nginxInfo.ConfPath, nginxInfo.Prefix),
			InstanceType: nginxType,
			Version:      version,
		},
		InstanceRuntime: instanceRuntime,
	}
}

func parseNginxVersionCommandOutput(ctx context.Context, output *bytes.Buffer) *Info {
	nginxInfo := &Info{}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "nginx version"):
			nginxInfo.Version = parseNginxVersion(line)
		case strings.HasPrefix(line, "configure arguments"):
			nginxInfo.ConfigureArgs = parseConfigureArguments(line)
		}
	}

	nginxInfo.Prefix = getNginxPrefix(ctx, nginxInfo)
	nginxInfo.ConfPath = getNginxConfPath(ctx, nginxInfo)

	return nginxInfo
}

func parseNginxVersion(line string) string {
	return strings.TrimPrefix(versionRegex.FindString(line), "nginx/")
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
	if mp, ok := nginxInfo.ConfigureArgs["modules-path"]; ok {
		modulePath, pathOK := mp.(string)
		if !pathOK {
			slog.Warn("Error parsing modules-path")
			return modules
		}
		modules, err = readDirectory(modulePath, ".so")
		if err != nil {
			slog.Warn("Error reading module dir", "dir", modulePath, "error", err)
			return modules
		}

		return modules
	}

	return modules
}

func getDynamicModules(nginxInfo *Info) (modules []string) {
	configArgs := nginxInfo.ConfigureArgs
	for arg := range configArgs {
		if strings.HasPrefix(arg, withWithPrefix) && strings.HasSuffix(arg, withModuleSuffix) {
			modules = append(modules, strings.TrimPrefix(arg, withWithPrefix))
		}
	}

	return modules
}

// readDirectory returns a list of all files in the directory which match the extension
func readDirectory(dir, extension string) (files []string, err error) {
	dirInfo, err := os.ReadDir(dir)
	if err != nil {
		return files, fmt.Errorf("read directory %s, %w", dir, err)
	}

	for _, file := range dirInfo {
		files = append(files, strings.ReplaceAll(file.Name(), extension, ""))
	}

	return files, err
}

func filterNginxProcesses(processes []*model.Process) map[int32]*model.Process {
	nginxProcesses := make(map[int32]*model.Process)

	for _, p := range processes {
		if isNginxProcess(p.Name, p.Cmd) {
			nginxProcesses[p.PID] = p
		}
	}

	return nginxProcesses
}

func isNginxProcess(name, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}
