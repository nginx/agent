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
)

// MockMessagePipe is a mock message pipe
type MockMessagePipe struct {
	plugins           []Plugin
	extensionPlugins  []ExtensionPlugin
	messages          []*Message
	processedMessages []*Message
	ctx               context.Context
}

var _ MessagePipeInterface = &MockMessagePipe{}

func SetupMockMessagePipe(t *testing.T, ctx context.Context, plugins []Plugin, extensionPlugins []ExtensionPlugin) *MockMessagePipe {
	messagePipe := NewMockMessagePipe(ctx)

	err := messagePipe.Register(10, plugins, extensionPlugins)
	if err != nil {
		t.Fail()
	}
	return messagePipe
}

func ValidateMessages(t *testing.T, messagePipe *MockMessagePipe, msgTopics []string) {
	processedMessages := messagePipe.GetProcessedMessages()
	if len(processedMessages) != len(msgTopics) {
		t.Fatalf("expected %d messages, received %d: %+v", len(msgTopics), len(processedMessages), processedMessages)
	}
	for idx, msg := range processedMessages {
		if msgTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), msgTopics[idx])
		}
	}
	messagePipe.ClearMessages()
}

func NewMockMessagePipe(ctx context.Context) *MockMessagePipe {
	return &MockMessagePipe{
		ctx: ctx,
	}
}

func (p *MockMessagePipe) Register(size int, plugins []Plugin, extensionPlugins []ExtensionPlugin) error {
	p.plugins = append(p.plugins, plugins...)
	p.extensionPlugins = append(p.extensionPlugins, extensionPlugins...)
	return nil
}

func (p *MockMessagePipe) Context() context.Context {
	return p.ctx
}

func (p *MockMessagePipe) Process(msgs ...*Message) {
	p.messages = append(p.messages, msgs...)
}

func (p *MockMessagePipe) GetMessages() []*Message {
	return p.messages
}

func (p *MockMessagePipe) GetProcessedMessages() []*Message {
	return p.processedMessages
}

func (p *MockMessagePipe) ClearMessages() {
	p.processedMessages = []*Message{}
	p.messages = []*Message{}
}

func (p *MockMessagePipe) Run() {
	for _, plugin := range p.plugins {
		plugin.Init(p)
	}

	for _, r := range p.extensionPlugins {
		r.Init(p)
	}

	p.RunWithoutInit()
}

func (p *MockMessagePipe) RunWithoutInit() {
	var message *Message
	for len(p.messages) > 0 {
		message, p.messages = p.messages[0], p.messages[1:]
		for _, plugin := range p.plugins {
			plugin.Process(message)
		}
		for _, plugin := range p.extensionPlugins {
			plugin.Process(message)
		}
		p.processedMessages = append(p.processedMessages, message)
	}
}

func (p *MockMessagePipe) GetPlugins() []Plugin {
	return p.plugins
}

func (p *MockMessagePipe) GetExtensionPlugins() []ExtensionPlugin {
	return p.extensionPlugins
}
