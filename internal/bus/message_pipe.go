/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package bus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/vardius/message-bus"
)

type Payload interface{}

type Message struct {
	Topic string
	Data  Payload
}

type Info struct {
	Name string
}

type Plugin interface {
	Init(MessagePipeInterface)
	Close()
	Info() *Info
	Process(*Message)
	Subscriptions() []string
}

//nolint:unused
type MessagePipeInterface interface {
	Register(int, []Plugin) error
	DeRegister(plugins []string) error
	Process(...*Message)
	Run()
	Context() context.Context
	GetPlugins() []Plugin
	IsPluginAlreadyRegistered(string) bool
}

type MessagePipe struct {
	bus            messagebus.MessageBus
	messageChannel chan *Message
	plugins        []Plugin
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewMessagePipe(ctx context.Context, size int) *MessagePipe {
	pipeContext, pipeCancel := context.WithCancel(ctx)
	return &MessagePipe{
		messageChannel: make(chan *Message, size),
		mu:             sync.RWMutex{},
		ctx:            pipeContext,
		cancel:         pipeCancel,
	}
}

func (p *MessagePipe) Register(size int, plugins []Plugin) error {
	p.mu.Lock()

	p.plugins = append(p.plugins, plugins...)
	p.bus = messagebus.New(size)

	pluginsRegistered := []string{}

	for _, plugin := range p.plugins {
		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Subscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
		pluginsRegistered = append(pluginsRegistered, plugin.Info().Name)
	}

	slog.Info("Finished registering plugins", "plugins", pluginsRegistered)

	p.mu.Unlock()
	return nil
}

func (p *MessagePipe) DeRegister(pluginNames []string) error {
	p.mu.Lock()

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

			for _, subscription := range plugin.Subscriptions() {
				err := p.bus.Unsubscribe(subscription, plugin.Process)
				if err != nil {
					return err
				}
			}
		}
	}

	p.mu.Unlock()
	return nil
}

func getIndex(pluginName string, plugins []Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name {
			return index
		}
	}
	return -1
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

			close(p.messageChannel)

			return
		case m := <-p.messageChannel:
			p.mu.Lock()
			p.bus.Publish(m.Topic, m)
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

func (p *MessagePipe) initPlugins() {
	for _, r := range p.plugins {
		r.Init(p)
	}
}

func (p *MessagePipe) IsPluginAlreadyRegistered(pluginName string) bool {
	pluginAlreadyRegistered := false
	for _, plugin := range p.GetPlugins() {
		if plugin.Info().Name == pluginName {
			pluginAlreadyRegistered = true
		}
	}
	return pluginAlreadyRegistered
}
