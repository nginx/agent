package bus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testPlugin struct {
	mock.Mock
}

func (p *testPlugin) Init(pipe *MessagePipe) {
	p.Called()
}

func (p *testPlugin) Process(message *Message) {
	p.Called()
}

func (p *testPlugin) Close() {
	p.Called()
}

func (p *testPlugin) Info() *Info {
	return &Info{"test"}
}

func (p *testPlugin) Subscriptions() []string {
	return []string{"test.message"}
}

func TestMessagePipe(t *testing.T) {
	messages := []*Message{
		{"test.message", 1},
		{"test.message", 2},
		{"test.message", 3},
		{"test.message", 4},
		{"test.message", 5},
	}

	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Process").Times(len(messages))
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(ctx, 100)
	err := messagePipe.Register(10, []Plugin{plugin})

	assert.NoError(t, err)

	go func() {
		messagePipe.Run()
		pipelineDone <- true
	}()

	messagePipe.Process(messages...)
	time.Sleep(10 * time.Millisecond) // for the above call being asynchronous

	cancel()
	<-pipelineDone

	plugin.AssertExpectations(t)
}

func TestMessagePipe_DeRegister(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagePipe := NewMessagePipe(ctx, 100)
	err := messagePipe.Register(100, []Plugin{plugin})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(messagePipe.GetPlugins()))

	err = messagePipe.DeRegister([]string{plugin.Info().Name})

	assert.NoError(t, err)
	assert.Equal(t, 0, len(messagePipe.GetPlugins()))
	plugin.AssertExpectations(t)
}

func TestMessagePipe_IsPluginAlreadyRegistered(t *testing.T) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := NewMessagePipe(ctx, 100)
	err := messagePipe.Register(10, []Plugin{plugin})

	assert.NoError(t, err)

	go func() {
		messagePipe.Run()
		pipelineDone <- true
	}() // for the above call being asynchronous

	cancel()
	<-pipelineDone

	assert.True(t, messagePipe.IsPluginAlreadyRegistered(plugin.Info().Name))
	assert.False(t, messagePipe.IsPluginAlreadyRegistered("metrics"))
}
