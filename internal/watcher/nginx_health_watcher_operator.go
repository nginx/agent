// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
)

type NginxHealthWatcher struct {
	executer exec.ExecInterface
}

var _ healthWatcherOperator = (*NginxHealthWatcher)(nil)

func NewNginxHealthWatcher() *NginxHealthWatcher {
	return &NginxHealthWatcher{
		executer: &exec.Exec{},
	}
}

func (nhw *NginxHealthWatcher) Health(ctx context.Context, instanceID string) *v1.InstanceHealth {
	return &v1.InstanceHealth{
		InstanceId:           instanceID,
		Description:          "instance is healthy",
		InstanceHealthStatus: 1,
	}
}
