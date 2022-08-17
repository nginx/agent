package core

import (
	"context"
	"testing"
)

// MockMessagePipe is a mock message pipe
type MockMessagePipe struct {
	plugins           []Plugin
	messages          []*Message
	processedMessages []*Message
	ctx               context.Context
}

var _ MessagePipeInterface = &MockMessagePipe{}

func SetupMockMessagePipe(t *testing.T, ctx context.Context, plugin ...Plugin) *MockMessagePipe {
	messagePipe := NewMockMessagePipe(ctx)

	err := messagePipe.Register(10, plugin...)
	if err != nil {
		t.Fail()
	}
	return messagePipe
}

func ValidateMessages(t *testing.T, messagePipe *MockMessagePipe, msgTopics []string) {
	processedMessages := messagePipe.GetProcessedMessages()
	if len(processedMessages) != len(msgTopics) {
		t.Fatalf("expected %d messages, received %d: %+v", len(msgTopics), len(processedMessages), processedMessages)
	}
	for idx, msg := range processedMessages {
		if msgTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), msgTopics[idx])
		}
	}
	messagePipe.ClearMessages()
}

func NewMockMessagePipe(ctx context.Context) *MockMessagePipe {
	return &MockMessagePipe{
		ctx: ctx,
	}
}

func (p *MockMessagePipe) Register(size int, plugin ...Plugin) error {
	p.plugins = append(p.plugins, plugin...)
	return nil
}

func (p *MockMessagePipe) Context() context.Context {
	return p.ctx
}

func (p *MockMessagePipe) Process(msgs ...*Message) {
	p.messages = append(p.messages, msgs...)
}

func (p *MockMessagePipe) GetMessages() []*Message {
	return p.messages
}

func (p *MockMessagePipe) GetProcessedMessages() []*Message {
	return p.processedMessages
}

func (p *MockMessagePipe) ClearMessages() {
	p.processedMessages = []*Message{}
	p.messages = []*Message{}
}

func (p *MockMessagePipe) Run() {
	for _, plugin := range p.plugins {
		plugin.Init(p)
	}
	p.RunWithoutInit()
}

func (p *MockMessagePipe) RunWithoutInit() {
	var message *Message
	for len(p.messages) > 0 {
		message, p.messages = p.messages[0], p.messages[1:]
		for _, plugin := range p.plugins {
			plugin.Process(message)
		}
		p.processedMessages = append(p.processedMessages, message)
	}
}

func (p *MockMessagePipe) GetPlugins() []Plugin {
	return p.plugins
}
