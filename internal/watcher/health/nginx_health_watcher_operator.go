// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"fmt"
	"strings"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/watcher/process"
)

type NginxHealthWatcher struct {
	executer        exec.ExecInterface
	processOperator process.ProcessOperatorInterface
}

var _ healthWatcherOperator = (*NginxHealthWatcher)(nil)

func NewNginxHealthWatcher() *NginxHealthWatcher {
	return &NginxHealthWatcher{
		executer:        &exec.Exec{},
		processOperator: process.NewProcessOperator(),
	}
}

func (nhw *NginxHealthWatcher) Health(ctx context.Context, instance *mpi.Instance) (*mpi.InstanceHealth, error) {
	proc, err := nhw.processOperator.Process(ctx, instance.GetInstanceRuntime().GetProcessId())
	if err != nil {
		return nil, err
	}

	health := &mpi.InstanceHealth{
		InstanceId:           instance.GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
	}
	if strings.Contains(proc.Status, "zombie") || !proc.Running {
		health.Description = fmt.Sprintf("PID: %d is degraded, status: %s", proc.PID, proc.Status)
		health.InstanceHealthStatus = mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY
	}

	if len(instance.GetInstanceRuntime().GetInstanceChildren()) == 0 {
		health.Description = fmt.Sprintf("%s, instance does not have enough children", health.GetDescription())
		health.InstanceHealthStatus = mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_DEGRADED
	}

	return health, nil
}
