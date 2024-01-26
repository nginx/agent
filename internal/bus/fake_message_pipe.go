/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package bus

import (
	"context"
	"testing"
)

// FakeMessagePipe is a mock message pipe
type FakeMessagePipe struct {
	plugins           []Plugin
	messages          []*Message
	processedMessages []*Message
	ctx               context.Context
}

var _ MessagePipeInterface = &FakeMessagePipe{}

func SetupFakeMessagePipe(t *testing.T, ctx context.Context, plugins []Plugin) *FakeMessagePipe {
	messagePipe := NewFakeMessagePipe(ctx)

	err := messagePipe.Register(10, plugins)
	if err != nil {
		t.Fail()
	}
	return messagePipe
}

func ValidateMessages(t *testing.T, messagePipe *FakeMessagePipe, msgTopics []string) {
	processedMessages := messagePipe.GetProcessedMessages()
	if len(processedMessages) != len(msgTopics) {
		t.Fatalf("expected %d messages, received %d: %+v", len(msgTopics), len(processedMessages), processedMessages)
	}
	for idx, msg := range processedMessages {
		if msgTopics[idx] != msg.Topic {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic, msgTopics[idx])
		}
	}
	messagePipe.ClearMessages()
}

func NewFakeMessagePipe(ctx context.Context) *FakeMessagePipe {
	return &FakeMessagePipe{
		ctx: ctx,
	}
}

func (p *FakeMessagePipe) Register(size int, plugins []Plugin) error {
	p.plugins = append(p.plugins, plugins...)
	return nil
}

func (p *FakeMessagePipe) DeRegister(pluginNames []string) error {
	var plugins []Plugin
	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	for _, plugin := range plugins {
		index := getIndex(plugin.Info().Name, p.plugins)

		if index != -1 {
			p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)

			plugin.Close()
		}

	}

	return nil
}

func (p *FakeMessagePipe) Context() context.Context {
	return p.ctx
}

func (p *FakeMessagePipe) Process(msgs ...*Message) {
	p.messages = append(p.messages, msgs...)
}

func (p *FakeMessagePipe) GetMessages() []*Message {
	return p.messages
}

func (p *FakeMessagePipe) GetProcessedMessages() []*Message {
	return p.processedMessages
}

func (p *FakeMessagePipe) ClearMessages() {
	p.processedMessages = []*Message{}
	p.messages = []*Message{}
}

func (p *FakeMessagePipe) Run() {
	for _, plugin := range p.plugins {
		plugin.Init(p)
	}

	p.RunWithoutInit()
}

func (p *FakeMessagePipe) RunWithoutInit() {
	var message *Message
	for len(p.messages) > 0 {
		message, p.messages = p.messages[0], p.messages[1:]
		for _, plugin := range p.plugins {
			plugin.Process(message)
		}
		p.processedMessages = append(p.processedMessages, message)
	}
}

func (p *FakeMessagePipe) GetPlugins() []Plugin {
	return p.plugins
}

func (p *FakeMessagePipe) IsPluginAlreadyRegistered(pluginName string) bool {
	pluginAlreadyRegistered := false
	for _, plugin := range p.GetPlugins() {
		if plugin.Info().Name == pluginName {
			pluginAlreadyRegistered = true
		}
	}
	return pluginAlreadyRegistered
}
