// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
)

type InstanceOperator struct {
	executer exec.ExecInterface
}

var _ instanceOperator = (*InstanceOperator)(nil)

func NewInstanceOperator() *InstanceOperator {
	return &InstanceOperator{
		executer: &exec.Exec{},
	}
}

func (i *InstanceOperator) Validate(ctx context.Context, instance *mpi.Instance) error {
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

func (i *InstanceOperator) validateConfigCheckResponse(out []byte) error {
	if bytes.Contains(out, []byte("[emerg]")) ||
		bytes.Contains(out, []byte("[alert]")) ||
		bytes.Contains(out, []byte("[crit]")) {
		return fmt.Errorf("error running nginx -t -c:\n%s", out)
	}

	return nil
}

func (i *InstanceOperator) Reload(ctx context.Context, instance *mpi.Instance) error {
	slog.InfoContext(ctx, "Reloading NGINX PID: ", "pid",
		instance.GetInstanceRuntime().GetProcessId())

	err := i.executer.KillProcess(instance.GetInstanceRuntime().GetProcessId())
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "NGINX reloaded", "processid", instance.GetInstanceRuntime().GetProcessId())

	return nil
}
