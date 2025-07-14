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
		Data  Payload
		Topic string
	}

	MessageWithContext struct {
		ctx     context.Context
		message *Message
	}

	Info struct {
		Name string
	}

	MessagePipeInterface interface {
		Register(size int, plugins []Plugin) error
		DeRegister(ctx context.Context, plugins []string) error
		Process(ctx context.Context, messages ...*Message)
		Run(ctx context.Context)
		Plugins() []Plugin
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
		messageChannel chan *MessageWithContext
		plugins        []Plugin
		pluginsMutex   sync.Mutex
	}
)

func NewMessagePipe(size int) *MessagePipe {
	return &MessagePipe{
		messageChannel: make(chan *MessageWithContext, size),
		pluginsMutex:   sync.Mutex{},
	}
}

func (p *MessagePipe) Register(size int, plugins []Plugin) error {
	p.pluginsMutex.Lock()
	defer p.pluginsMutex.Unlock()

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
	p.pluginsMutex.Lock()
	defer p.pluginsMutex.Unlock()

	plugins := p.findPlugins(pluginNames)

	for _, plugin := range plugins {
		index := p.Index(plugin.Info().Name, p.plugins)

		err := p.unsubscribePlugin(ctx, index, plugin)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *MessagePipe) Process(ctx context.Context, messages ...*Message) {
	for _, message := range messages {
		p.messageChannel <- &MessageWithContext{ctx, message}
	}
}

func (p *MessagePipe) Run(ctx context.Context) {
	p.pluginsMutex.Lock()
	p.initPlugins(ctx)
	p.pluginsMutex.Unlock()

	for {
		select {
		case <-ctx.Done():
			p.pluginsMutex.Lock()
			for _, r := range p.plugins {
				r.Close(ctx)
			}
			p.pluginsMutex.Unlock()

			return
		case m := <-p.messageChannel:
			p.bus.Publish(m.message.Topic, m.ctx, m.message)
		}
	}
}

func (p *MessagePipe) Plugins() []Plugin {
	return p.plugins
}

func (p *MessagePipe) IsPluginRegistered(pluginName string) bool {
	isPluginRegistered := false

	for _, plugin := range p.Plugins() {
		if plugin.Info().Name == pluginName {
			isPluginRegistered = true
		}
	}

	return isPluginRegistered
}

func (p *MessagePipe) Index(pluginName string, plugins []Plugin) int {
	for index, plugin := range plugins {
		if pluginName == plugin.Info().Name {
			return index
		}
	}

	return -1
}

func (p *MessagePipe) unsubscribePlugin(ctx context.Context, index int, plugin Plugin) error {
	if index != -1 {
		p.plugins = append(p.plugins[:index], p.plugins[index+1:]...)

		err := plugin.Close(ctx)
		if err != nil {
			return err
		}

		for _, subscription := range plugin.Subscriptions() {
			unsubErr := p.bus.Unsubscribe(subscription, plugin.Process)
			if unsubErr != nil {
				return unsubErr
			}
		}
	}

	return nil
}

func (p *MessagePipe) findPlugins(pluginNames []string) []Plugin {
	var plugins []Plugin

	for _, name := range pluginNames {
		for _, plugin := range p.plugins {
			if plugin.Info().Name == name {
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins
}

func (p *MessagePipe) initPlugins(ctx context.Context) {
	for index, plugin := range p.plugins {
		err := plugin.Init(ctx, p)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to initialize plugin", "plugin", plugin.Info().Name, "error", err)

			unsubscribeError := p.unsubscribePlugin(ctx, index, plugin)
			if unsubscribeError != nil {
				slog.ErrorContext(
					ctx,
					"Failed to unsubscribe plugin",
					"plugin", plugin.Info().Name,
					"error", unsubscribeError,
				)
			}
		}
	}
}
