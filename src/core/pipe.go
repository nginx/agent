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
	messagebus "github.com/vardius/message-bus"
)

const (
	MessageQueueSize    = 100
	MaxPlugins          = 50
	MaxExtensionPlugins = 50
)

type MessagePipeInterface interface {
	Register(int, []Plugin, []ExtensionPlugin) error
	Process(...*Message)
	Run()
	Context() context.Context
	GetPlugins() []Plugin
	GetExtensionPlugins() []ExtensionPlugin
}

type MessagePipe struct {
	messageChannel   chan *Message
	plugins          []Plugin
	extensionPlugins []ExtensionPlugin
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.Mutex
	bus              messagebus.MessageBus
}

func NewMessagePipe(ctx context.Context) *MessagePipe {
	pipeContext, pipeCancel := context.WithCancel(ctx)
	return &MessagePipe{
		messageChannel:   make(chan *Message, MessageQueueSize),
		plugins:          make([]Plugin, 0, MaxPlugins),
		extensionPlugins: make([]ExtensionPlugin, 0, MaxExtensionPlugins),
		ctx:              pipeContext,
		cancel:           pipeCancel,
		mu:               sync.Mutex{},
	}
}

func (p *MessagePipe) Register(size int, plugins []Plugin, extensionPlugins []ExtensionPlugin) error {
	p.mu.Lock()

	p.plugins = append(p.plugins, plugins...)
	p.extensionPlugins = append(p.extensionPlugins, extensionPlugins...)
	p.bus = messagebus.New(size)

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

	log.Infof("The following core plugins have being registered: %q", pluginsRegistered)
	log.Infof("The following extension plugins have being registered: %q", extensionPluginsRegistered)

	p.mu.Unlock()
	return nil
}

func (p *MessagePipe) Process(messages ...*Message) {
	for _, m := range messages {
		select {
		case p.messageChannel <- m:
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *MessagePipe) Run() {
	p.initPlugins()

	for {
		select {
		case <-p.ctx.Done():
			for _, r := range p.plugins {
				r.Close()
			}

			for _, r := range p.extensionPlugins {
				r.Close()
			}

			close(p.messageChannel)

			return
		case m := <-p.messageChannel:
			p.mu.Lock()
			p.bus.Publish(m.Topic(), m)
			p.mu.Unlock()
		}
	}
}

func (p *MessagePipe) Context() context.Context {
	return p.ctx
}

func (p *MessagePipe) Cancel() context.CancelFunc {
	return p.cancel
}

func (p *MessagePipe) GetPlugins() []Plugin {
	return p.plugins
}

func (p *MessagePipe) GetExtensionPlugins() []ExtensionPlugin {
	return p.extensionPlugins
}

func (p *MessagePipe) initPlugins() {
	for _, r := range p.plugins {
		r.Init(p)
	}

	for _, r := range p.extensionPlugins {
		r.Init(p)
	}
}
