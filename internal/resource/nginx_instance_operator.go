// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/pkg/id"

	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/pkg/nginxprocess"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
)

type NginxInstanceOperator struct {
	agentConfig           *config.Config
	executer              exec.ExecInterface
	logTailer             logTailerOperator
	nginxProccessOperator processOperator
	treatWarningsAsErrors bool
}

var _ instanceOperator = (*NginxInstanceOperator)(nil)

func NewInstanceOperator(agentConfig *config.Config) *NginxInstanceOperator {
	return &NginxInstanceOperator{
		executer:              &exec.Exec{},
		logTailer:             NewLogTailerOperator(agentConfig),
		nginxProccessOperator: NewNginxInstanceProcessOperator(),
		treatWarningsAsErrors: agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsErrors,
		agentConfig:           agentConfig,
	}
}

func (i *NginxInstanceOperator) Validate(ctx context.Context, instance *mpi.Instance) error {
	slog.DebugContext(ctx, "Validating NGINX config")
	exePath := instance.GetInstanceRuntime().GetBinaryPath()

	out, err := i.executer.RunCmd(ctx, exePath, "-t")
	if err != nil {
		return fmt.Errorf("NGINX config test failed %w: %s", err, out)
	}

	err = i.validateConfigCheckResponse(out.Bytes())
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "NGINX config tested", "output", out)

	return nil
}

func (i *NginxInstanceOperator) Reload(ctx context.Context, instance *mpi.Instance) error {
	var reloadTime time.Time
	var errorsFound error
	slog.InfoContext(ctx, "Reloading NGINX PID", "pid",
		instance.GetInstanceRuntime().GetProcessId())

	pid := instance.GetInstanceRuntime().GetProcessId()
	workers := i.nginxWorkerProcesses(ctx, pid)

	if len(workers) > 0 {
		reloadTime = workers[0].Created
	}

	errorLogs := i.errorLogs(instance)

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go i.monitorLogs(ctx, errorLogs, logErrorChannel)

	err := i.executer.KillProcess(instance.GetInstanceRuntime().GetProcessId())
	if err != nil {
		return err
	}

	processes, procErr := i.nginxProccessOperator.FindNginxProcesses(ctx)
	if procErr != nil {
		slog.WarnContext(ctx, "Error finding parent process ID, unable to check if NGINX worker "+
			"processes have reloaded", "error", procErr)
	} else {
		i.checkWorkers(ctx, instance.GetInstanceMeta().GetInstanceId(), reloadTime, processes)
	}

	slog.InfoContext(ctx, "NGINX reloaded", "process_id", instance.GetInstanceRuntime().GetProcessId())

	numberOfExpectedMessages := len(errorLogs)

	for range numberOfExpectedMessages {
		logErr := <-logErrorChannel
		slog.InfoContext(ctx, "Message received in logErrorChannel", "error", logErr)
		if logErr != nil {
			errorsFound = errors.Join(errorsFound, logErr)
			slog.InfoContext(ctx, "Errors Found", "", errorsFound)
		}
	}

	slog.InfoContext(ctx, "Finished monitoring post reload")

	if errorsFound != nil {
		return errorsFound
	}

	return nil
}

func (i *NginxInstanceOperator) checkWorkers(ctx context.Context, instanceID string, reloadTime time.Time,
	processes []*nginxprocess.Process,
) {
	backoffSettings := &config.BackOff{
		InitialInterval:     i.agentConfig.Client.Backoff.InitialInterval,
		MaxInterval:         i.agentConfig.Client.Backoff.MaxInterval,
		MaxElapsedTime:      i.agentConfig.Client.Backoff.MaxElapsedTime,
		RandomizationFactor: i.agentConfig.Client.Backoff.RandomizationFactor,
		Multiplier:          i.agentConfig.Client.Backoff.Multiplier,
	}

	slog.DebugContext(ctx, "Waiting for NGINX to finish reloading")
	newPid, findErr := i.findParentProcessID(ctx, instanceID, processes)
	slog.InfoContext(ctx, "ppid", "", newPid)

	if findErr != nil {
		slog.WarnContext(ctx, "Error finding parent process ID, unable to check if NGINX worker "+
			"processes have reloaded", "error", findErr)

		return
	}

	err := backoff.WaitUntil(ctx, backoffSettings, func() error {
		currentWorkers := i.nginxWorkerProcesses(ctx, newPid)
		if len(currentWorkers) == 0 {
			return errors.New("waiting for NGINX worker processes")
		}

		for _, worker := range currentWorkers {
			if !worker.Created.After(reloadTime) {
				return fmt.Errorf("waiting for all NGINX workers to be newer "+
					"than %v, found worker with time %v", reloadTime, worker.Created)
			}
		}

		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, "Failed to check if NGINX worker processes have successfully reloaded, "+
			"timed out waiting", "error", err)

		return
	}

	slog.InfoContext(ctx, "All NGINX workers have been reloaded")
}

func (i *NginxInstanceOperator) nginxWorkerProcesses(ctx context.Context, pid int32) []*nginxprocess.Process {
	slog.DebugContext(ctx, "Getting NGINX worker processes for NGINX reload")
	var workers []*nginxprocess.Process
	nginxProcesses, err := i.nginxProccessOperator.FindNginxProcesses(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get NGINX processes", "error", err)
		return workers
	}

	for _, nginxProcess := range nginxProcesses {
		if nginxProcess.IsWorker() && nginxProcess.PPID == pid {
			workers = append(workers, nginxProcess)
		}
	}

	return workers
}

func (i *NginxInstanceOperator) findParentProcessID(ctx context.Context, instanceID string,
	nginxProcesses []*nginxprocess.Process,
) (int32, error) {
	var pid int32

	for _, proc := range nginxProcesses {
		if proc.IsMaster() {
			info, infoErr := nginx.ProcessInfo(ctx, proc, i.executer)
			if infoErr != nil {
				slog.WarnContext(ctx, "Failed to get NGINX process info from master process", "error", infoErr)
				continue
			}
			processInstanceID := id.Generate("%s_%s_%s", info.ExePath, info.ConfPath, info.Prefix)
			if instanceID == processInstanceID {
				slog.DebugContext(ctx, "Found NGINX process ID", "process_id", processInstanceID)
				return proc.PID, nil
			}
		}
	}

	return pid, errors.New("unable to find parent process")
}

func (i *NginxInstanceOperator) validateConfigCheckResponse(out []byte) error {
	if bytes.Contains(out, []byte("[emerg]")) ||
		bytes.Contains(out, []byte("[alert]")) ||
		bytes.Contains(out, []byte("[crit]")) {
		return fmt.Errorf("error running nginx -t -c:\n%s", out)
	}

	if i.treatWarningsAsErrors && bytes.Contains(out, []byte("[warn]")) {
		return fmt.Errorf("error running nginx -t -c:\n%s", out)
	}

	return nil
}

func (i *NginxInstanceOperator) errorLogs(instance *mpi.Instance) (errorLogs []string) {
	if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		errorLogs = instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetErrorLogs()
	} else if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX {
		errorLogs = instance.GetInstanceRuntime().GetNginxRuntimeInfo().GetErrorLogs()
	}

	return errorLogs
}

func (i *NginxInstanceOperator) monitorLogs(ctx context.Context, errorLogs []string, errorChannel chan error) {
	if len(errorLogs) == 0 {
		slog.InfoContext(ctx, "No NGINX error logs found to monitor")
		return
	}

	for _, errorLog := range errorLogs {
		go i.logTailer.Tail(ctx, errorLog, errorChannel)
	}
}
