// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package bus

import (
	"context"
	"testing"
	"time"

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

func TestMessagePipe(t *testing.T) {
	messages := []*Message{
		{"test.message", 1},
		{"test.message", 2},
		{"test.message", 3},
		{"test.message", 4},
		{"test.message", 5},
	}

	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Process").Times(len(messages))
	plugin.On("Close", mock.Anything).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(100)
	err := messagePipe.Register(10, []Plugin{plugin})

	require.NoError(t, err)

	go func() {
		messagePipe.Run(ctx)
		pipelineDone <- true
	}()

	messagePipe.Process(messages...)
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

	messagePipe := NewMessagePipe(100)
	err := messagePipe.Register(100, []Plugin{plugin})

	require.NoError(t, err)
	assert.Len(t, messagePipe.GetPlugins(), 1)

	err = messagePipe.DeRegister(ctx, []string{plugin.Info().Name})

	require.NoError(t, err)
	assert.Empty(t, len(messagePipe.GetPlugins()))
	plugin.AssertExpectations(t)
}

func TestMessagePipe_IsPluginRegistered(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(100)
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
