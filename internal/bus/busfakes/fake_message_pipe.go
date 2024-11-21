// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package busfakes

import (
	"context"
	"github.com/nginx/agent/v3/internal/bus"
	"sync"
)

// FakeMessagePipe is a mock message pipe
type FakeMessagePipe struct {
	plugins           []bus.Plugin
	messages          []*bus.Message
	processedMessages []*bus.Message
	messagesLock      sync.Mutex
}

var _ bus.MessagePipeInterface = &FakeMessagePipe{}

func NewFakeMessagePipe() *FakeMessagePipe {
	return &FakeMessagePipe{
		messagesLock: sync.Mutex{},
	}
}

func (p *FakeMessagePipe) Register(size int, plugins []bus.Plugin) error {
	p.plugins = append(p.plugins, plugins...)
	return nil
}

func (p *FakeMessagePipe) DeRegister(ctx context.Context, pluginNames []string) error {
	var plugins []bus.Plugin

	plugins = p.findPlugins(pluginNames, plugins)

	for _, plugin := range plugins {
		index := p.GetIndex(plugin.Info().Name, p.plugins)
		p.unsubscribePlugin(ctx, index, plugin)
	}

	return nil
}

func (p *FakeMessagePipe) GetIndex(pluginName string, plugins []bus.Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name {
			return index
		}
	}

	return -1
}

func (p *FakeMessagePipe) unsubscribePlugin(ctx context.Context, index int, plugin bus.Plugin) {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)
		err := plugin.Close(ctx)
		if err != nil {
			return
		}
	}
}

func (p *FakeMessagePipe) findPlugins(pluginNames []string, plugins []bus.Plugin) []bus.Plugin {
	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins
}

func (p *FakeMessagePipe) Process(_ context.Context, msgs ...*bus.Message) {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	p.messages = append(p.messages, msgs...)
}

func (p *FakeMessagePipe) GetMessages() []*bus.Message {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	return p.messages
}

func (p *FakeMessagePipe) GetProcessedMessages() []*bus.Message {
	return p.processedMessages
}

func (p *FakeMessagePipe) ClearMessages() {
	p.messagesLock.Lock()
	defer p.messagesLock.Unlock()

	p.processedMessages = []*bus.Message{}
	p.messages = []*bus.Message{}
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
	var message *bus.Message

	for len(p.messages) > 0 {
		message, p.messages = p.messages[0], p.messages[1:]
		for _, plugin := range p.plugins {
			plugin.Process(ctx, message)
		}
		p.processedMessages = append(p.processedMessages, message)
	}
}

func (p *FakeMessagePipe) GetPlugins() []bus.Plugin {
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
