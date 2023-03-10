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
		NewMessage("test.message", 2),
		NewMessage("test.message", 3),
		NewMessage("test.message", 4),
		NewMessage("test.message", 5),
	}

	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Process").Times(len(messages))
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(ctx)
	err := messagePipe.Register(10, []Plugin{plugin}, nil)

	assert.NoError(t, err)

	go func() {
		messagePipe.Run()
		pipelineDone <- true
	}()

	messagePipe.Process(messages...)
	time.Sleep(10 * time.Millisecond) // for the above call being asynchronous

	cancel()
	<-pipelineDone

	plugin.AssertExpectations(t)
}
