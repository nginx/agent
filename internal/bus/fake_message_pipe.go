// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

import (
	"context"
)

// FakeMessagePipe is a mock message pipe
type FakeMessagePipe struct {
	plugins           []Plugin
	messages          []*Message
	processedMessages []*Message
	ctx               context.Context
}

var _ MessagePipeInterface = &FakeMessagePipe{}

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

	plugins = p.findPlugins(pluginNames, plugins)

	for _, plugin := range plugins {
		index := getIndex(plugin.Info().Name, p.plugins)
		p.unsubscribePlugin(index, plugin)
	}

	return nil
}

func (p *FakeMessagePipe) unsubscribePlugin(index int, plugin Plugin) {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)
		plugin.Close()
	}
}

func (p *FakeMessagePipe) findPlugins(pluginNames []string, plugins []Plugin) []Plugin {
	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins
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
