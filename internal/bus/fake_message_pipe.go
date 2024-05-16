// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

import (
	"context"
	"sync"
)

// FakeMessagePipe is a mock message pipe
type FakeMessagePipe struct {
	plugins           []Plugin
	messages          []*Message
	processedMessages []*Message
	messagesLock      sync.Mutex
}

var _ MessagePipeInterface = &FakeMessagePipe{}

func NewFakeMessagePipe() *FakeMessagePipe {
	return &FakeMessagePipe{
		messagesLock: sync.Mutex{},
	}
}

func (p *FakeMessagePipe) Register(size int, plugins []Plugin) error {
	p.plugins = append(p.plugins, plugins...)
	return nil
}

func (p *FakeMessagePipe) DeRegister(ctx context.Context, pluginNames []string) error {
	var plugins []Plugin

	plugins = p.findPlugins(pluginNames, plugins)

	for _, plugin := range plugins {
		index := getIndex(plugin.Info().Name, p.plugins)
		p.unsubscribePlugin(ctx, index, plugin)
	}

	return nil
}

func (p *FakeMessagePipe) unsubscribePlugin(ctx context.Context, index int, plugin Plugin) {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)
		plugin.Close(ctx)
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

func (p *FakeMessagePipe) Process(ctx context.Context, msgs ...*Message) {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	p.messages = append(p.messages, msgs...)
}

func (p *FakeMessagePipe) GetMessages() []*Message {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	return p.messages
}

func (p *FakeMessagePipe) GetProcessedMessages() []*Message {
	return p.processedMessages
}

func (p *FakeMessagePipe) ClearMessages() {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	p.processedMessages = []*Message{}
	p.messages = []*Message{}
}

func (p *FakeMessagePipe) Run(ctx context.Context) {
	for _, plugin := range p.plugins {
		err := plugin.Init(ctx, p)
		if err != nil {
			return
		}
	}

	p.RunWithoutInit(ctx)
}

func (p *FakeMessagePipe) RunWithoutInit(ctx context.Context) {
	var message *Message

	// p.messagesLock.Lock()
	// defer p.messagesLock.Unlock()

	for len(p.messages) > 0 {
		message, p.messages = p.messages[0], p.messages[1:]
		for _, plugin := range p.plugins {
			plugin.Process(ctx, message)
		}
		p.processedMessages = append(p.processedMessages, message)
	}
}

func (p *FakeMessagePipe) GetPlugins() []Plugin {
	return p.plugins
}

func (p *FakeMessagePipe) IsPluginRegistered(pluginName string) bool {
	pluginAlreadyRegistered := false

	for _, plugin := range p.GetPlugins() {
		if plugin.Info().Name == pluginName {
			pluginAlreadyRegistered = true
		}
	}

	return pluginAlreadyRegistered
}
