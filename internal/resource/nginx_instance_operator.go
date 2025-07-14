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
	var errorsFound error
	slog.InfoContext(ctx, "Reloading NGINX PID", "pid",
		instance.GetInstanceRuntime().GetProcessId())

	slog.InfoContext(ctx, "NGINX reloaded", "processid", instance.GetInstanceRuntime().GetProcessId())

	errorLogs := i.errorLogs(instance)

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go i.monitorLogs(ctx, errorLogs, logErrorChannel)

	err := i.executer.KillProcess(instance.GetInstanceRuntime().GetProcessId())
	if err != nil {
		return err
	}

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
