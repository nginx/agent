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
	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
	"log/slog"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
)

type NginxInstanceOperator struct {
	executer              exec.ExecInterface
	logTailer             logTailerOperator
	treatWarningsAsErrors bool
}

var _ instanceOperator = (*NginxInstanceOperator)(nil)

func NewInstanceOperator(agentConfig *config.Config) *NginxInstanceOperator {
	return &NginxInstanceOperator{
		executer:              &exec.Exec{},
		logTailer:             NewLogTailerOperator(agentConfig),
		treatWarningsAsErrors: agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsErrors,
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

	workers := nginxWorkerProcesses(ctx)

	if workers != nil && len(workers) > 0 {
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

	backoffSettings := &config.BackOff{
		InitialInterval:     config.DefBackoffInitialInterval,
		MaxInterval:         config.DefBackoffMaxInterval,
		MaxElapsedTime:      config.DefBackoffMaxElapsedTime,
		RandomizationFactor: config.DefBackoffRandomizationFactor,
		Multiplier:          config.DefBackoffMultiplier,
	}

	slog.Info("Waiting for NGINX to finish reloading")
	err = backoff.WaitUntil(ctx, backoffSettings, func() error {
		currentWorkers := nginxWorkerProcesses(ctx)
		if currentWorkers == nil || len(currentWorkers) == 0 {
			return fmt.Errorf("waiting for NGINX worker processes")
		}

		for _, worker := range currentWorkers {
			if !worker.Created.After(reloadTime) {
				return fmt.Errorf("waiting for all NGINX workers to be newer "+
					"than %v, found worker with time %v", reloadTime, worker.Created)
			}
		}

		slog.InfoContext(ctx, "All NGINX workers have been reloaded", "worker_count", len(currentWorkers))
		return nil
	})

	slog.InfoContext(ctx, "NGINX reloaded", "processid", instance.GetInstanceRuntime().GetProcessId())

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

func nginxWorkerProcesses(ctx context.Context) []*nginxprocess.Process {
	slog.Debug("Getting NGINX worker processes for NGINX reload")
	var workers []*nginxprocess.Process
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		slog.Warn("Failed to get processes", "error", err)
	}

	nginxProcesses, err := nginxprocess.ListWithProcesses(ctx, processes)

	if err != nil {
		slog.Warn("Failed to get NGINX processes", "error", err)
	}

	for _, nginxProcess := range nginxProcesses {
		if nginxProcess.IsWorker() {
			workers = append(workers, nginxProcess)
		}
	}

	return workers
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
