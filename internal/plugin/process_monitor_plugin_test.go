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
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestProcessMonitor_Init(t *testing.T) {
	testProcesses := []*model.Process{{Pid: 123, Name: "nginx"}}

	processMonitor := NewProcessMonitor(&config.Config{
		ProcessMonitor: config.ProcessMonitor{
			MonitoringFrequency: time.Millisecond,
		},
	})

	processMonitor.getProcessesFunc = func() ([]*model.Process, error) {
		return testProcesses, nil
	}

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{processMonitor})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, processMonitor.messagePipe)
	assert.Equal(t, testProcesses, processMonitor.processes)
}

func TestProcessMonitor_Info(t *testing.T) {
	processMonitor := NewProcessMonitor(&config.Config{})
	info := processMonitor.Info()
	assert.Equal(t, "process-monitor", info.Name)
}

func TestProcessMonitor_Subscriptions(t *testing.T) {
	processMonitor := NewProcessMonitor(&config.Config{})
	subscriptions := processMonitor.Subscriptions()
	assert.Equal(t, []string{}, subscriptions)
}
