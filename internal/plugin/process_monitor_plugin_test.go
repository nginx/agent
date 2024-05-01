// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestProcessMonitor_Init(t *testing.T) {
	ctx := context.Background()
	testProcesses := map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}}
	processMonitor := NewProcessMonitor(types.GetAgentConfig())

	processMonitor.getProcessesFunc = func(_ context.Context) (map[int32]*model.Process, error) {
		return testProcesses, nil
	}

	messagePipe := bus.NewMessagePipe(100)
	err := messagePipe.Register(100, []bus.Plugin{processMonitor})
	require.NoError(t, err)
	go messagePipe.Run(ctx)

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testProcesses, processMonitor.getProcesses())
}

func TestProcessMonitor_Info(t *testing.T) {
	processMonitor := NewProcessMonitor(types.GetAgentConfig())
	info := processMonitor.Info()
	assert.Equal(t, "process-monitor", info.Name)
}

func TestProcessMonitor_Subscriptions(t *testing.T) {
	processMonitor := NewProcessMonitor(types.GetAgentConfig())
	subscriptions := processMonitor.Subscriptions()
	assert.Equal(t, []string{}, subscriptions)
}

func TestProcessMonitor_haveProcessesChanged(t *testing.T) {
	tests := []struct {
		name         string
		oldProcesses map[int32]*model.Process
		newProcesses map[int32]*model.Process
		expected     bool
	}{
		{
			name:         "Test 1: number of processes are the same and PIDs have not changed",
			oldProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			newProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			expected:     false,
		},
		{
			name:         "Test 2: number of processes are the same but PIDs are different",
			oldProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			newProcesses: map[int32]*model.Process{456: {Pid: 456, Name: "nginx"}},
			expected:     true,
		},
		{
			name:         "Test 3: number of new processes is less than old processes",
			oldProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			newProcesses: make(map[int32]*model.Process),
			expected:     true,
		},
		{
			name:         "Test 4: number of new processes is more than old processes",
			oldProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			newProcesses: map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}, 456: {Pid: 456, Name: "nginx"}},
			expected:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := haveProcessesChanged(test.oldProcesses, test.newProcesses)
			assert.Equal(tt, test.expected, actual)
		})
	}
}
