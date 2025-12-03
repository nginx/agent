// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package bus

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testPlugin struct {
	mock.Mock
}

func (p *testPlugin) Init(_ context.Context, pipe MessagePipeInterface) error {
	p.Called()
	return nil
}

func (p *testPlugin) Process(_ context.Context, _ *Message) {
	p.Called()
}

func (p *testPlugin) Close(_ context.Context) error {
	p.Called()
	return nil
}

func (*testPlugin) Info() *Info {
	return &Info{"test"}
}

func (*testPlugin) Subscriptions() []string {
	return []string{"test.message"}
}

func (p *testPlugin) Reconfigure(ctx context.Context, agentConfig *config.Config) error {
	p.Called()
	return nil
}

func TestMessagePipe(t *testing.T) {
	messages := []*Message{
		{Topic: "test.message", Data: 1},
		{Topic: "test.message", Data: 2},
		{Topic: "test.message", Data: 3},
		{Topic: "test.message", Data: 4},
		{Topic: "test.message", Data: 5},
	}

	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Process").Times(len(messages))
	plugin.On("Close", mock.Anything).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(100, types.AgentConfig())
	err := messagePipe.Register(10, []Plugin{plugin})

	require.NoError(t, err)

	go func() {
		messagePipe.Run(ctx)
		pipelineDone <- true
	}()

	messagePipe.Process(ctx, messages...)
	time.Sleep(10 * time.Millisecond) // for the above call being asynchronous

	cancel()
	<-pipelineDone

	plugin.AssertExpectations(t)
}

func TestMessagePipe_DeRegister(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagePipe := NewMessagePipe(100, types.AgentConfig())
	err := messagePipe.Register(100, []Plugin{plugin})

	require.NoError(t, err)
	assert.Len(t, messagePipe.Plugins(), 1)

	err = messagePipe.DeRegister(ctx, []string{plugin.Info().Name})

	require.NoError(t, err)
	assert.Empty(t, messagePipe.Plugins())
	plugin.AssertExpectations(t)
}

func TestMessagePipe_IsPluginRegistered(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(100, types.AgentConfig())
	err := messagePipe.Register(10, []Plugin{plugin})

	require.NoError(t, err)

	go func() {
		messagePipe.Run(ctx)
		pipelineDone <- true
	}() // for the above call being asynchronous

	cancel()
	<-pipelineDone

	assert.True(t, messagePipe.IsPluginRegistered(plugin.Info().Name))
	assert.False(t, messagePipe.IsPluginRegistered("metrics"))
}

func TestMessagePipe_updateConfig(t *testing.T) {
	initialConfig := types.AgentConfig()
	initialConfig.Log.Level = "INFO"
	initialConfig.Log.Path = ""
	initialConfig.Labels = map[string]any{"old": "value"}

	messagePipe := NewMessagePipe(100, initialConfig)
	originalLogger := slog.Default()

	updatedConfig := &config.Config{
		Log: &config.Log{
			Path:  "/etc/nginx-agent/",
			Level: "DEBUG",
		},
		Labels: map[string]any{
			"version": "5.0",
			"test":    "config",
		},
	}

	messagePipe.updateConfig(context.Background(), updatedConfig)

	require.Equal(t, messagePipe.agentConfig.Log.Path, updatedConfig.Log.Path)
	require.Equal(t, messagePipe.agentConfig.Log.Level, updatedConfig.Log.Level)
	require.Equal(t, messagePipe.agentConfig.Labels, updatedConfig.Labels)

	newLogger := slog.Default()
	require.NotEqual(t, originalLogger, newLogger)
}
