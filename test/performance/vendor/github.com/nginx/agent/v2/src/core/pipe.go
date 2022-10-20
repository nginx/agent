package core

import (
	"context"
	"sync"

	messagebus "github.com/vardius/message-bus"
)

const (
	MessageQueueSize = 100
	MaxPlugins       = 100
)

type MessagePipeInterface interface {
	Register(int, ...Plugin) error
	Process(...*Message)
	Run()
	Context() context.Context
	GetPlugins() []Plugin
}

type MessagePipe struct {
	messageChannel chan *Message
	plugins        []Plugin
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	bus            messagebus.MessageBus
}

func NewMessagePipe(ctx context.Context) *MessagePipe {
	pipeContext, pipeCancel := context.WithCancel(ctx)
	return &MessagePipe{
		messageChannel: make(chan *Message, MessageQueueSize),
		plugins:        make([]Plugin, 0, MaxPlugins),
		ctx:            pipeContext,
		cancel:         pipeCancel,
		mu:             sync.Mutex{},
	}
}

func (p *MessagePipe) Register(size int, plugins ...Plugin) error {
	p.mu.Lock()

	p.plugins = append(p.plugins, plugins...)
	p.bus = messagebus.New(size)	

	for _, plugin := range p.plugins {
		for _, subscription := range plugin.Subscriptions() {
			err := p.bus.Subscribe(subscription, plugin.Process)
			if err != nil {
				return err
			}
		}
	}

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

func (p *MessagePipe) initPlugins() {
	for _, r := range p.plugins {
		r.Init(p)
	}
}
