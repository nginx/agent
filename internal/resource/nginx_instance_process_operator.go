// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"

	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
)

type NginxInstanceProcessOperator struct{}

var _ processOperator = (*NginxInstanceProcessOperator)(nil)

func NewNginxInstanceProcessOperator() *NginxInstanceProcessOperator {
	return &NginxInstanceProcessOperator{}
}

func (p *NginxInstanceProcessOperator) FindNginxProcesses(ctx context.Context) ([]*nginxprocess.Process, error) {
	processes, procErr := process.ProcessesWithContext(ctx)
	if procErr != nil {
		return nil, procErr
	}

	nginxProcesses, err := nginxprocess.ListWithProcesses(ctx, processes)
	if err != nil {
		return nil, err
	}

	return nginxProcesses, nil
}
