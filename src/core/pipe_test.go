/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testPlugin struct {
	mock.Mock
}

func (p *testPlugin) Init(pipe MessagePipeInterface) {
	p.Called()
}

func (p *testPlugin) Process(message *Message) {
	p.Called()
}

func (p *testPlugin) Close() {
	p.Called()
}

func (p *testPlugin) Info() *Info {
	return NewInfo("test", "v0.0.1")
}

func (p *testPlugin) Subscriptions() []string {
	return []string{"test.message"}
}

func TestMessagePipe(t *testing.T) {
	messages := []*Message{
		NewMessage("test.message", 1),
	}

	plugin := new(testPlugin)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())

	messagePipe := NewMessagePipe(ctx, 100)
	err := messagePipe.Register(10, []Plugin{plugin}, nil)

	require.NoError(t, err)

	messagePipe.Process(messages...)

	select {
	case msg := <-messagePipe.messageChannel:
		assert.Equal(t, "test.message", *msg.topic)
	case <-time.After(time.Second):
		t.Fatal("Expected message not received")
	}

	err = messagePipe.DeRegister([]string{*plugin.Info().name})
	require.NoError(t, err)

	plugin.AssertExpectations(t)
	cancel()
	time.Sleep(1 * time.Second) // Allow some time for the message to be processed
}

func TestMessagePipe_Run(t *testing.T) {
	messages := []*Message{
		NewMessage("test.message", 1),
		NewMessage("test.message", 2),
		NewMessage("test.message", 3),
		NewMessage("test.message", 4),
		NewMessage("test.message", 5),
	}

	ctx, cancel := context.WithCancel(context.Background())

	pipe := NewMessagePipe(ctx, 10)

	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
    plugin.On("Process").Times(len(messages))
	plugin.On("Close").Times(1)

	pipe.Register(10, []Plugin{plugin}, nil)

	go pipe.Run()

	pipe.Process(messages...)

	time.Sleep(100 * time.Millisecond) // Allow some time for the message to be processed
	cancel()

	time.Sleep(1 * time.Second) // Allow some time for the message to be processed

}

func TestMessagePipe_Process(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pipe := NewMessagePipe(ctx, 10)

	messages := []*Message{
		NewMessage("test.message", 1),
	}

	pipe.Process(messages...)

	select {
	case msg := <-pipe.messageChannel:
		assert.Equal(t, "test.message", *msg.topic)
	case <-time.After(time.Second):
		t.Fatal("Expected message not received")
	}
}

func TestPipe_DeRegister(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagePipe := NewMessagePipe(ctx, 10)
	messagePipe.Register(10, []Plugin{plugin}, nil)

	err := messagePipe.DeRegister([]string{*plugin.Info().name})
	require.NoError(t, err)

	assert.Equal(t, 0, len(messagePipe.GetPlugins()))
}

func TestPipe_IsPluginAlreadyRegistered(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pipelineDone := make(chan bool)
	defer close(pipelineDone)

	messagePipe := NewMessagePipe(ctx, 100)
	err := messagePipe.Register(10, []Plugin{plugin}, nil)

	require.NoError(t, err)

	assert.True(t, messagePipe.IsPluginAlreadyRegistered(*plugin.Info().name))
	assert.False(t, messagePipe.IsPluginAlreadyRegistered("metrics"))
	
	err = messagePipe.DeRegister([]string{*plugin.Info().name})
	require.NoError(t, err)
}
