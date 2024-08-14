/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	message_bus "github.com/vardius/message-bus"
)

const (
	MaxPlugins          = 50
	MaxExtensionPlugins = 50
)

type MessagePipeInterface interface {
	Register(int, []Plugin, []ExtensionPlugin) error
	DeRegister(plugins []string) error
	Process(...*Message)
	Run()
	Context() context.Context
	GetPlugins() []Plugin
	GetExtensionPlugins() []ExtensionPlugin
	IsPluginAlreadyRegistered(string) bool
	Close()
}

type MessagePipe struct {
	messageChannel   chan *Message
	plugins          []Plugin
	extensionPlugins []ExtensionPlugin
	ctx              context.Context
	mu               sync.RWMutex
	bus              message_bus.MessageBus
}

func NewMessagePipe(ctx context.Context, size int) *MessagePipe {
	return &MessagePipe{
		messageChannel:   make(chan *Message, size),
		plugins:          make([]Plugin, 0, MaxPlugins),
		extensionPlugins: make([]ExtensionPlugin, 0, MaxExtensionPlugins),
		ctx:              ctx,
		mu:               sync.RWMutex{},
		bus:              message_bus.New(size),
	}
}

func InitializePipe(ctx context.Context, corePlugins []Plugin, extensionPlugins []ExtensionPlugin, size int) MessagePipeInterface {
	pipe := NewMessagePipe(ctx, size)
	err := pipe.Register(size, corePlugins, extensionPlugins)
	if err != nil {
		log.Warnf("Failed to start agent successfully, error loading plugins %v", err)
	}
	return pipe
}

func (p *MessagePipe) Register(size int, plugins []Plugin, extensionPlugins []ExtensionPlugin) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.plugins = append(p.plugins, plugins...)
	p.extensionPlugins = append(p.extensionPlugins, extensionPlugins...)

	pluginsRegistered := []string{}
	extensionPluginsRegistered := []string{}

	for _, plugin := range p.plugins {
		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Subscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
		pluginsRegistered = append(pluginsRegistered, *plugin.Info().name)
	}

	for _, plugin := range p.extensionPlugins {
		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Subscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
		extensionPluginsRegistered = append(extensionPluginsRegistered, *plugin.Info().name)
	}
	log.Infof("The following core plugins have been registered: %q", pluginsRegistered)
	log.Infof("The following extension plugins have been registered: %q", extensionPluginsRegistered)

	return nil
}

func (p *MessagePipe) DeRegister(pluginNames []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var pluginsToRemove []Plugin
	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name() == name {
				pluginsToRemove = append(pluginsToRemove, plugin)
			}
		}
	}

	for _, plugin := range pluginsToRemove {
		index := getIndex(plugin.Info().Name(), p.plugins)

		if index != -1 {
			p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)

			plugin.Close()

			for _, subscription := range plugin.Subscriptions() {
				err := p.bus.Unsubscribe(subscription, plugin.Process)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *MessagePipe) Close() {
	p.cleanup()
}

func getIndex(pluginName string, plugins []Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name() {
			return index
		}
	}
	return -1
}

func (p *MessagePipe) Process(messages ...*Message) {
	for _, m := range messages {
		select {
		case <-p.ctx.Done():
			p.cleanup()
			return
		case p.messageChannel <- m:
		default:
			return
		}
	}
}

func (p *MessagePipe) Run() {
	p.initPlugins()

	for {
		select {
		case <-p.ctx.Done():
			p.cleanup()
			return
		case m := <-p.messageChannel:
			p.mu.Lock()
			if p.bus != nil {
				p.bus.Publish(m.Topic(), m)
			}
			p.mu.Unlock()
		}
	}
}

func (p *MessagePipe) Context() context.Context {
	return p.ctx
}

func (p *MessagePipe) GetPlugins() []Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.plugins
}

func (p *MessagePipe) GetExtensionPlugins() []ExtensionPlugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.extensionPlugins
}

func (p *MessagePipe) cleanup() {
	for _, r := range p.plugins {
		for _, subscription := range r.Subscriptions() {
			p.bus.Close(subscription)
		}
		r.Close()
	}

	for _, r := range p.extensionPlugins {
		for _, subscription := range r.Subscriptions() {
			p.bus.Close(subscription)
		}
		r.Close()
	}

	p.bus = nil
	p.plugins = nil
	if p.messageChannel != nil {
		close(p.messageChannel)
	}
	p.messageChannel = nil
}

func (p *MessagePipe) initPlugins() {
	for _, r := range p.plugins {
		r.Init(p)
	}

	for _, r := range p.extensionPlugins {
		r.Init(p)
	}
}

func (p *MessagePipe) IsPluginAlreadyRegistered(pluginName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, plugin := range p.GetPlugins() {
		if plugin.Info().Name() == pluginName {
			return true
		}
	}
	return false
}
