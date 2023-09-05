package performance

import (
	"context"
	"testing"
	"time"

	sdk "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/plugins"
	tutils "github.com/nginx/agent/v2/test/utils"

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

func BenchmarkFeaturesExtensionsAndPlugins(b *testing.B) {
	detailsMap := tutils.GetDetailsMap()
	procMap := tutils.GetProcessMap()

	binary := tutils.GetMockNginxBinary()
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(detailsMap)
	binary.On("GetNginxIDForProcess", mock.Anything).Return("12345")
	binary.On("UpdateNginxDetailsFromProcesses", mock.Anything).Return()
	binary.On("GetChildProcesses").Return(procMap)
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)

	env := tutils.GetMockEnvWithHostAndProcess()
	env.Mock.On("IsContainer").Return(false)
	env.Mock.On("DiskDevices").Return([]string{"disk1", "disk2"}, nil)
	env.Mock.On("GetNetOverflow").Return(0.0, nil)

	tests := []struct {
		name                  string
		loadedConfig          *config.Config
		expectedPluginSize    int
		expectedExtensionSize int
	}{
		// {
		// 	name: "default plugins",
		// 	loadedConfig: &config.Config{
		// 		Server: tutils.GetMockAgentConfig().Server,
		// 	},
		// 	expectedPluginSize:    5,
		// 	expectedExtensionSize: 0,
		// },
		// {
		// 	name: "no plugins or extensions",
		// 	loadedConfig: &config.Config{
		// 		Version:    "v9.99.99999",
		// 		Server:     tutils.GetMockAgentConfig().Server,
		// 		Features:   []string{},
		// 		Extensions: []string{},
		// 	},
		// 	expectedPluginSize:    5,
		// 	expectedExtensionSize: 0,
		// },
		{
			name: "all plugins and extensions",
			loadedConfig: &config.Config{
				Version:  "v9.99.999",
				Server:   tutils.GetMockAgentConfig().Server,
				Features: sdk.GetDefaultFeatures(),
				// temporarily to figure out what's going on with the monitoring extension
				Extensions: sdk.GetKnownExtensions()[:len(sdk.GetKnownExtensions())-1],
				AgentMetrics: config.AgentMetrics{
					BulkSize:           1,
					ReportInterval:     1,
					CollectionInterval: 1,
					Mode:               "aggregated",
				},
				AgentAPI: config.AgentAPI{
					Port: 2345,
					Key:  "",
					Cert: "",
				},
				Dataplane: config.Dataplane{
					Status: config.Status{PollInterval: 60 * time.Second},
				},
			},
			expectedPluginSize: 15,
			// temporarily to figure out what's going on with the monitoring extension
			expectedExtensionSize: len(sdk.GetKnownExtensions()[:len(sdk.GetKnownExtensions())-1]),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(t *testing.B) {
			for i := 0; i < b.N; i++ {
				b.ResetTimer()
				ctx, cancel := context.WithCancel(context.Background())
				pipelineDone := make(chan bool)

				controller, cmdr, reporter := core.CreateGrpcClients(ctx, tt.loadedConfig)
				corePlugins, extensionPlugins := plugins.LoadPlugins(cmdr, binary, env, reporter, tt.loadedConfig)

				pipe := core.InitializePipe(ctx, corePlugins, extensionPlugins, 20)
				core.HandleSignals(ctx, cmdr, tt.loadedConfig, env, pipe, cancel, controller)

				event := events.NewAgentEventMeta(config.MODULE,
					tt.loadedConfig.Version,
					"1234",
					"Initialize Agent",
					env.GetHostname(),
					env.GetSystemUUID(),
					tt.loadedConfig.InstanceGroup,
					tt.loadedConfig.Tags)

				pipe.Process(core.NewMessage(core.AgentStarted, event))

				go func() {
					pipe.Run()
					pipelineDone <- true
				}()

				<-pipelineDone

				assert.NotNil(t, corePlugins)
				assert.Len(t, corePlugins, tt.expectedPluginSize)
				assert.Len(t, extensionPlugins, tt.expectedExtensionSize)

				controller.Close()
				ctx.Done()
			}
		})
	}
}

func BenchmarkPluginOneTimeRegistration(b *testing.B) {
	var pluginsUnderTest []core.Plugin

	ctx, cancel := context.WithCancel(context.Background())
	pipelineDone := make(chan bool)

	config := config.Config{
		Nginx:   config.Nginx{Debug: true},
		Version: "1234",
	}

	processID := "12345"
	detailsMap := map[string]*proto.NginxDetails{
		processID: {
			ProcessPath: "/path/to/nginx",
			NginxId:     processID,
		},
	}

	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(detailsMap)
	binary.On("GetNginxIDForProcess", mock.Anything).Return(processID)
	binary.On("GetNginxDetailsFromProcess", mock.Anything).Return(detailsMap[processID])
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)

	env := tutils.GetMockEnvWithHostAndProcess()
	meta := proto.Metadata{}

	messagePipe := core.NewMessagePipe(ctx)
	for n := 0; n < b.N; n++ {
		pluginsUnderTest = append(pluginsUnderTest, plugins.NewOneTimeRegistration(&config, binary, env, &meta))
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
