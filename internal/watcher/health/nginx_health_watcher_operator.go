// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"fmt"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	processwatcher "github.com/nginx/agent/v3/internal/watcher/process"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
)

type NginxHealthWatcher struct {
	executer        exec.ExecInterface
	processOperator processwatcher.ProcessOperatorInterface
}

var _ healthWatcherOperator = (*NginxHealthWatcher)(nil)

func NewNginxHealthWatcher() *NginxHealthWatcher {
	return &NginxHealthWatcher{
		executer:        &exec.Exec{},
		processOperator: processwatcher.NewProcessOperator(),
	}
}

func (nhw *NginxHealthWatcher) Health(ctx context.Context, instance *mpi.Instance) (*mpi.InstanceHealth, error) {
	health := &mpi.InstanceHealth{
		InstanceId:           instance.GetInstanceMeta().GetInstanceId(),
		InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
	}

	proc, err := nhw.processOperator.Process(ctx, instance.GetInstanceRuntime().GetProcessId())
	if nginxprocess.IsNotRunningErr(err) {
		health.Description = fmt.Sprintf("PID: %d is not running", instance.GetInstanceRuntime().GetProcessId())
		health.InstanceHealthStatus = mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY

		return health, err
	} else if err != nil {
		return nil, err
	}

	if !proc.IsHealthy() {
		health.Description = fmt.Sprintf("PID: %d is unhealthy, status: %s", proc.PID, proc.Status)
		health.InstanceHealthStatus = mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY
	}

	if len(instance.GetInstanceRuntime().GetInstanceChildren()) == 0 {
		health.Description = health.GetDescription() + ", instance does not have enough children"
		health.InstanceHealthStatus = mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_DEGRADED
	}

	return health, nil
}
