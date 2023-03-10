package performance

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/plugins"
	"github.com/nginx/agent/v2/test/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testPlugin struct {
	mock.Mock
}

func (p *testPlugin) Init(pipe core.MessagePipeInterface) {
	p.Called()
}

func (p *testPlugin) Process(message *core.Message) {
	p.Called()
}

func (p *testPlugin) Close() {
	p.Called()
}

func (p *testPlugin) Info() *core.Info {
	return core.NewInfo("test", "v0.0.1")
}

func (p *testPlugin) Subscriptions() []string {
	return []string{"test.message"}
}

func BenchmarkPlugin(b *testing.B) {
	plugin := new(testPlugin)
	plugin.On("Init").Times(1)
	plugin.On("Process").Times(b.N)
	plugin.On("Close").Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	messagePipe := core.NewMessagePipe(ctx)
	err := messagePipe.Register(b.N, []core.Plugin{plugin}, []core.ExtensionPlugin{})
	assert.NoError(b, err)

	go func() {
		messagePipe.Run()
		pipelineDone <- true
	}()

	for n := 0; n < b.N; n++ {
		messagePipe.Process(core.NewMessage("test.message", n))
		time.Sleep(200 * time.Millisecond) // for the above call being asynchronous
	}

	cancel()
	<-pipelineDone

	plugin.AssertExpectations(b)
}

func BenchmarkPluginOneTimeRegistration(b *testing.B) {
	var pluginsUnderTest []core.Plugin

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	config := config.Config{Nginx: config.Nginx{Debug: true}}

	processID := "12345"
	detailsMap := map[string]*proto.NginxDetails{
		processID: {
			ProcessPath: "/path/to/nginx",
			NginxId:     processID,
		},
	}

	binary := utils.NewMockNginxBinary()
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(detailsMap)
	binary.On("GetNginxIDForProcess", mock.Anything).Return(processID)
	binary.On("GetNginxDetailsFromProcess", mock.Anything).Return(detailsMap[processID])
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)

	env := utils.NewMockEnvironment()
	env.Mock.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})
	env.Mock.On("Processes", mock.Anything).Return([]core.Process{
		{
			Name:     processID,
			IsMaster: true,
		},
	})

	meta := proto.Metadata{}
	version := "1234"

	messagePipe := core.NewMessagePipe(ctx)
	for n := 0; n < b.N; n++ {
		pluginsUnderTest = append(pluginsUnderTest, plugins.NewOneTimeRegistration(&config, binary, env, &meta, version))
	}

	err := messagePipe.Register(b.N, pluginsUnderTest, []core.ExtensionPlugin{})
	assert.NoError(b, err)

	go func() {
		messagePipe.Run()
		pipelineDone <- true
	}()

	for n := 0; n < b.N; n++ {
		messagePipe.Process(core.NewMessage(core.UNKNOWN, n))
		time.Sleep(200 * time.Millisecond) // for the above call being asynchronous
	}

	cancel()
	<-pipelineDone
}
