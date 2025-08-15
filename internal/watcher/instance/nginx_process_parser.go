// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/model"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/pkg/host/exec"
	"github.com/nginx/agent/v3/pkg/id"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
)

const (
	withWithPrefix   = "with-"
	withModuleSuffix = "module"
	keyValueLen      = 2
	flagLen          = 1
)

type (
	NginxProcessParser struct {
		executer exec.ExecInterface
	}
)

var _ processParser = (*NginxProcessParser)(nil)

func NewNginxProcessParser() *NginxProcessParser {
	return &NginxProcessParser{
		executer: &exec.Exec{},
	}
}

// cognitive complexity of 16 because of the if statements in the for loop
// don't think can be avoided due to the need for continue
// nolint: revive
func (npp *NginxProcessParser) Parse(ctx context.Context, processes []*nginxprocess.Process) map[string]*mpi.Instance {
	instanceMap := make(map[string]*mpi.Instance)   // key is instanceID
	workers := make(map[int32][]*mpi.InstanceChild) // key is ppid of process

	processesByPID := convertToMap(processes)

	for _, proc := range processesByPID {
		if proc.IsWorker() {
			// Here we are determining if the worker process has a master
			if masterProcess, ok := processesByPID[proc.PPID]; ok {
				workers[masterProcess.PID] = append(workers[masterProcess.PID],
					&mpi.InstanceChild{ProcessId: proc.PID})

				continue
			}
			nginxInfo, err := npp.info(ctx, proc)
			if err != nil {
				slog.DebugContext(ctx, "Unable to get NGINX info", "pid", proc.PID, "error", err)

				continue
			}
			// set instance process ID to 0 as there is no master process
			nginxInfo.ProcessID = 0

			instance := convertInfoToInstance(*nginxInfo)

			if foundInstance, ok := instanceMap[instance.GetInstanceMeta().GetInstanceId()]; ok {
				foundInstance.GetInstanceRuntime().InstanceChildren = append(foundInstance.GetInstanceRuntime().
					GetInstanceChildren(), &mpi.InstanceChild{ProcessId: proc.PID})

				continue
			}

			instance.GetInstanceRuntime().InstanceChildren = append(instance.GetInstanceRuntime().
				GetInstanceChildren(), &mpi.InstanceChild{ProcessId: proc.PID})

			instanceMap[instance.GetInstanceMeta().GetInstanceId()] = instance

			continue
		}

		// check if proc is a master process, process is not a worker but could be cache manager etc
		if proc.IsMaster() {
			nginxInfo, err := npp.info(ctx, proc)
			if err != nil {
				slog.DebugContext(ctx, "Unable to get NGINX info", "pid", proc.PID, "error", err)

				continue
			}

			instance := convertInfoToInstance(*nginxInfo)
			instanceMap[instance.GetInstanceMeta().GetInstanceId()] = instance
		}
	}

	for _, instance := range instanceMap {
		if val, ok := workers[instance.GetInstanceRuntime().GetProcessId()]; ok {
			instance.InstanceRuntime.InstanceChildren = val
		}
	}

	return instanceMap
}

func (npp *NginxProcessParser) info(ctx context.Context, proc *nginxprocess.Process) (*model.ProcessInfo, error) {
	nginxInfo, err := nginx.ProcessInfo(ctx, proc, npp.executer)
	if err != nil {
		return nil, err
	}
	loadableModules := loadableModules(nginxInfo)
	nginxInfo.LoadableModules = loadableModules

	nginxInfo.DynamicModules = dynamicModules(nginxInfo)

	return nginxInfo, err
}

func convertInfoToInstance(nginxInfo model.ProcessInfo) *mpi.Instance {
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
					StubStatus: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
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
					StubStatus: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: nginxInfo.LoadableModules,
					DynamicModules:  nginxInfo.DynamicModules,
					PlusApi: &mpi.APIDetails{
						Location: "",
						Listen:   "",
					},
				},
			},
		}

		nginxType = mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS
		version = nginxInfo.Version
	}

	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   id.Generate("%s_%s_%s", nginxInfo.ExePath, nginxInfo.ConfPath, nginxInfo.Prefix),
			InstanceType: nginxType,
			Version:      version,
		},
		InstanceRuntime: instanceRuntime,
	}
}

func loadableModules(nginxInfo *model.ProcessInfo) (modules []string) {
	var err error
	if mp, ok := nginxInfo.ConfigureArgs["modules-path"]; ok {
		modulePath, pathOK := mp.(string)
		if !pathOK {
			slog.Debug("Error parsing modules-path")
			return modules
		}
		modules, err = readDirectory(modulePath, ".so")
		if err != nil {
			slog.Debug("Error reading module dir", "dir", modulePath, "error", err)
			return modules
		}

		sort.Strings(modules)

		return modules
	}

	return modules
}

func dynamicModules(nginxInfo *model.ProcessInfo) (modules []string) {
	configArgs := nginxInfo.ConfigureArgs
	for arg := range configArgs {
		if strings.HasPrefix(arg, withWithPrefix) && strings.HasSuffix(arg, withModuleSuffix) {
			modules = append(modules, strings.TrimPrefix(arg, withWithPrefix))
		}
	}

	sort.Strings(modules)

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

func convertToMap(processes []*nginxprocess.Process) map[int32]*nginxprocess.Process {
	processesByPID := make(map[int32]*nginxprocess.Process)

	for _, p := range processes {
		processesByPID[p.PID] = p
	}

	return processesByPID
}
