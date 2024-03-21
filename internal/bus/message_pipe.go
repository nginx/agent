// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package bus

import (
	"context"
	"log/slog"
	"sync"

	messagebus "github.com/vardius/message-bus"
)

type (
	Payload interface{}

	Message struct {
		Topic string
		Data  Payload
	}

	Info struct {
		Name string
	}

	MessagePipeInterface interface {
		Register(size int, plugins []Plugin) error
		DeRegister(ctx context.Context, plugins []string) error
		Process(messages ...*Message)
		Run(ctx context.Context)
		GetPlugins() []Plugin
		IsPluginRegistered(pluginName string) bool
	}

	Plugin interface {
		Init(ctx context.Context, messagePipe MessagePipeInterface) error
		Close(ctx context.Context) error
		Info() *Info
		Process(ctx context.Context, msg *Message)
		Subscriptions() []string
	}

	MessagePipe struct {
		bus            messagebus.MessageBus
		messageChannel chan *Message
		plugins        []Plugin
		mu             sync.RWMutex
	}
)

func NewMessagePipe(size int) *MessagePipe {
	return &MessagePipe{
		messageChannel: make(chan *Message, size),
		mu:             sync.RWMutex{},
	}
}

func (p *MessagePipe) Register(size int, plugins []Plugin) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	return nil
}

func (p *MessagePipe) DeRegister(ctx context.Context, pluginNames []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var plugins []Plugin
	plugins = p.findPlugins(pluginNames, plugins)

	for _, plugin := range plugins {
		index := getIndex(plugin.Info().Name, p.plugins)

		err := p.unsubscribePlugin(ctx, index, plugin)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *MessagePipe) Process(messages ...*Message) {
	for _, m := range messages {
		p.messageChannel <- m
	}
}

func (p *MessagePipe) Run(ctx context.Context) {
	p.initPlugins(ctx)

	for {
		select {
		case <-ctx.Done():
			for _, r := range p.plugins {
				r.Close(ctx)
			}

			close(p.messageChannel)

			return
		case m := <-p.messageChannel:
			p.mu.Lock()
			p.bus.Publish(m.Topic, ctx, m)
			p.mu.Unlock()
		}
	}
}

func (p *MessagePipe) GetPlugins() []Plugin {
	return p.plugins
}

func (p *MessagePipe) IsPluginRegistered(pluginName string) bool {
	isPluginRegistered := false

	for _, plugin := range p.GetPlugins() {
		if plugin.Info().Name == pluginName {
			isPluginRegistered = true
		}
	}

	return isPluginRegistered
}

func (p *MessagePipe) unsubscribePlugin(ctx context.Context, index int, plugin Plugin) error {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)

		plugin.Close(ctx)

		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Unsubscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *MessagePipe) findPlugins(pluginNames []string, plugins []Plugin) []Plugin {
	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins
}

func getIndex(pluginName string, plugins []Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name {
			return index
		}
	}

	return -1
}

func (p *MessagePipe) initPlugins(ctx context.Context) {
	for _, r := range p.plugins {
		err := r.Init(ctx, p)
		if err != nil {
			slog.Error("Failed to initialize plugin", "plugin", r.Info().Name, "error", err)
		}
	}
}
