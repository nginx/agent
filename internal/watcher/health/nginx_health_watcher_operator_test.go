// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"fmt"
	"testing"

	"github.com/shirou/gopsutil/v4/process"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/watcher/process/processfakes"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNginxHealthWatcherOperator_Health(t *testing.T) {
	ctx := context.Background()
	nginxHealthWatcher := NewNginxHealthWatcher()
	fakeProcessOperator := &processfakes.FakeProcessOperatorInterface{}
	instance := protos.GetNginxOssInstance([]string{})
	noChildrenInstance := protos.GetNginxOssInstance([]string{})
	noChildrenInstance.GetInstanceRuntime().InstanceChildren = []*mpi.InstanceChild{}

	tests := []struct {
		expected *mpi.InstanceHealth
		instance *mpi.Instance
		process  *nginxprocess.Process
		err      error
		name     string
	}{
		{
			name: "Test 1: Healthy Instance",
			process: &nginxprocess.Process{
				PID:    instance.GetInstanceRuntime().GetProcessId(),
				PPID:   456,
				Name:   "nginx",
				Cmd:    "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
				Exe:    "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
				Status: "running",
			},
			expected: &mpi.InstanceHealth{
				InstanceId:           instance.GetInstanceMeta().GetInstanceId(),
				InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
			},
			err:      nil,
			instance: instance,
		},
		{
			name: "Test 2: Unhealthy Instance, Not Running",
			expected: &mpi.InstanceHealth{
				InstanceId: instance.GetInstanceMeta().GetInstanceId(),
				Description: fmt.Sprintf(
					"PID: %d is not running",
					instance.GetInstanceRuntime().GetProcessId(),
				),
				InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
			},
			err:      process.ErrorProcessNotRunning,
			instance: instance,
		},
		{
			name: "Test 3: Degraded Instance, Not Children",
			process: &nginxprocess.Process{
				PID:    instance.GetInstanceRuntime().GetProcessId(),
				PPID:   456,
				Name:   "nginx",
				Cmd:    "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
				Exe:    "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
				Status: "zombie",
			},
			expected: &mpi.InstanceHealth{
				InstanceId: instance.GetInstanceMeta().GetInstanceId(),
				Description: fmt.Sprintf(
					"PID: %d is unhealthy, status: %s, instance does not have enough children",
					instance.GetInstanceRuntime().GetProcessId(),
					"zombie",
				),
				InstanceHealthStatus: mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_DEGRADED,
			},
			err:      nil,
			instance: noChildrenInstance,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeProcessOperator.ProcessReturns(test.process, test.err)
			nginxHealthWatcher.processOperator = fakeProcessOperator

			instanceHealth, healthErr := nginxHealthWatcher.Health(ctx, test.instance)

			require.Equal(t, test.err, healthErr)
			assert.Equal(t, test.expected, instanceHealth)
			assert.True(t, proto.Equal(test.expected, instanceHealth))
		})
	}
}
