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

	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/pkg/nginxprocess"

	"github.com/nginx/agent/v3/pkg/host/exec"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
)

type NginxInstanceOperator struct {
	agentConfig           *config.Config
	executer              exec.ExecInterface
	logTailer             logTailerOperator
	nginxProcessOperator  processOperator
	treatWarningsAsErrors bool
}

var _ instanceOperator = (*NginxInstanceOperator)(nil)

func NewInstanceOperator(agentConfig *config.Config) *NginxInstanceOperator {
	return &NginxInstanceOperator{
		executer:              &exec.Exec{},
		logTailer:             NewLogTailerOperator(agentConfig),
		nginxProcessOperator:  NewNginxInstanceProcessOperator(),
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
	var createdTime time.Time
	var errorsFound error
	pid := instance.GetInstanceRuntime().GetProcessId()

	slog.InfoContext(ctx, "Reloading NGINX PID", "pid",
		pid)

	workers := i.nginxProcessOperator.NginxWorkerProcesses(ctx, pid)

	if len(workers) > 0 {
		createdTime = workers[0].Created
	}

	errorLogs := i.errorLogs(instance)

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go i.monitorLogs(ctx, errorLogs, logErrorChannel)

	err := i.executer.KillProcess(pid)
	if err != nil {
		return err
	}

	processes, procErr := i.nginxProcessOperator.FindNginxProcesses(ctx)
	if procErr != nil {
		slog.WarnContext(ctx, "Error finding parent process ID, unable to check if NGINX worker "+
			"processes have reloaded", "error", procErr)
	} else {
		i.checkWorkers(ctx, instance.GetInstanceMeta().GetInstanceId(), createdTime, processes)
	}

	slog.InfoContext(ctx, "NGINX reloaded", "process_id", pid)

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

func (i *NginxInstanceOperator) checkWorkers(ctx context.Context, instanceID string, createdTime time.Time,
	processes []*nginxprocess.Process,
) {
	backoffSettings := &config.BackOff{
		InitialInterval:     i.agentConfig.DataPlaneConfig.Nginx.ReloadBackoff.InitialInterval,
		MaxInterval:         i.agentConfig.DataPlaneConfig.Nginx.ReloadBackoff.MaxInterval,
		MaxElapsedTime:      i.agentConfig.DataPlaneConfig.Nginx.ReloadBackoff.MaxElapsedTime,
		RandomizationFactor: i.agentConfig.DataPlaneConfig.Nginx.ReloadBackoff.RandomizationFactor,
		Multiplier:          i.agentConfig.DataPlaneConfig.Nginx.ReloadBackoff.Multiplier,
	}

	slog.DebugContext(ctx, "Waiting for NGINX to finish reloading")

	newPid, findErr := i.nginxProcessOperator.FindParentProcessID(ctx, instanceID, processes, i.executer)
	if findErr != nil {
		slog.WarnContext(ctx, "Error finding parent process ID, unable to check if NGINX worker "+
			"processes have reloaded", "error", findErr)

		return
	}

	slog.DebugContext(ctx, "Found parent process ID, checking NGINX worker processes have reloaded",
		"process_id", newPid)

	err := backoff.WaitUntil(ctx, backoffSettings, func() error {
		currentWorkers := i.nginxProcessOperator.NginxWorkerProcesses(ctx, newPid)
		if len(currentWorkers) == 0 {
			return errors.New("waiting for NGINX worker processes")
		}

		for _, worker := range currentWorkers {
			if worker.Created.After(createdTime) {
				return nil
			}
		}

		return fmt.Errorf("waiting for NGINX worker to be newer "+
			"than %v", createdTime)
	})
	if err != nil {
		slog.WarnContext(ctx, "Failed to check if NGINX worker processes have successfully reloaded, "+
			"timed out waiting", "error", err)

		return
	}

	slog.InfoContext(ctx, "NGINX workers have been reloaded")
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
