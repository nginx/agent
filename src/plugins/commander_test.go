package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestCommander_Process(t *testing.T) {
	tests := []struct {
		name       string
		setMocks   bool
		cmd        *proto.Command
		topic      string
		nginxId    string
		systemId   string
		config     *proto.NginxConfig
		msgTopics  []string
		updateTags []string
	}{
		{
			name: "test agent connect",
			cmd: &proto.Command{
				Data: &proto.Command_AgentConnectResponse{
					AgentConnectResponse: &proto.AgentConnectResponse{
						AgentConfig: &proto.AgentConfig{
							Details: &proto.AgentDetails{
								Tags:       []string{"new-tag1:one", "new-tag2:two"},
								Extensions: []string{"advanced_metrics", "nginx_app_protect"},
							},
							Configs: &proto.ConfigReport{
								Configs: []*proto.ConfigDescriptor{
									{
										Checksum: "",
										NginxId:  "12345",
										SystemId: "6789",
									},
								},
							},
						},
					}},
			},
			setMocks:   false,
			topic:      core.AgentConnected,
			updateTags: []string{"new-tag1:one", "new-tag2:two"},
			nginxId:    "12345",
			systemId:   "6789",
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					SystemId: "6789",
					Checksum: "",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      []byte("nginx conf contents"),
					Checksum:      checksum.Checksum([]byte("nginx conf contents")),
					RootDirectory: "nginx.conf",
				},
				Zaux:         &proto.ZippedFile{},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.AgentConnected,
				core.AgentConfigChanged,
				core.NginxConfigUpload,
				core.EnableExtension,
				core.EnableExtension,
			},
		},
		{
			name: "test agent connect without config",
			cmd: &proto.Command{
				Data: &proto.Command_AgentConnectResponse{
					AgentConnectResponse: &proto.AgentConnectResponse{}},
			},
			topic:    core.AgentConnected,
			setMocks: false,
			nginxId:  "",
			systemId: "",
			config:   nil,
			msgTopics: []string{
				core.AgentConnected,
			},
		},
		{
			name: "test agent register",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: &proto.Command_AgentConnectRequest{
					AgentConnectRequest: &proto.AgentConnectRequest{
						Host: &proto.HostInfo{},
						Meta: &proto.AgentMeta{},
					},
				},
			},
			setMocks: true,
			topic:    core.CommRegister,
			nginxId:  "",
			systemId: "",
			config:   nil,
			msgTopics: []string{
				core.CommRegister,
			},
		},
		{
			name: "test agent config apply",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: &proto.Command_NginxConfigResponse{
					NginxConfigResponse: &proto.NginxConfigResponse{
						Status: newOKStatus("config applied successfully").CmdStatus,
						Action: proto.NginxConfigAction_APPLY,
						ConfigData: &proto.ConfigDescriptor{
							NginxId: "12345",
						},
					},
				},
			},
			topic:    core.CommNginxConfig,
			nginxId:  "12345",
			systemId: "67890",
			msgTopics: []string{
				core.CommNginxConfig,
			},
		},
		{
			name: "test agent config request",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: &proto.Command_AgentConfigRequest{
					AgentConfigRequest: &proto.AgentConfigRequest{},
				},
			},
			topic: core.AgentConfig,
			msgTopics: []string{
				core.AgentConfig,
			},
		},
		{
			name: "test agent command status ok",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: newOKStatus("ok"),
			},
			topic: core.UNKNOWN,
			msgTopics: []string{
				core.UNKNOWN,
			},
		},
		{
			name: "test agent command status not ok",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: newErrStatus("err"),
			},
			topic: core.UNKNOWN,
			msgTopics: []string{
				core.UNKNOWN,
			},
		},
		{
			name: "test agent command data nil",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: nil,
			},
			topic: core.UNKNOWN,
			msgTopics: []string{
				core.UNKNOWN,
			},
		},
		{
			name: "test agent command data nil",
			cmd: &proto.Command{
				Meta: &proto.Metadata{},
				Type: proto.Command_NORMAL,
				Data: newOKStatus("ok"),
			},
			topic: core.UNKNOWN,
			msgTopics: []string{
				core.UNKNOWN,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			// Create an agent config and initialize Viper config properties
			// based off of it, clean up when done.
			// TODO: The test agent config is going to be getting modified.
			// Need to either not run parallel or properly lock the code.
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			ctx := context.TODO()
			cmdr := tutils.NewMockCommandClient()
			mockChannel := testChannel(&proto.Command{})

			// setup expectations
			cmdr.On("Recv").Return(mockChannel)
			if test.setMocks {
				cmdr.On("Send", mock.Anything, client.MessageFromCommand(test.cmd))
			}

			pluginUnderTest := NewCommander(cmdr, &config.Config{ClientID: "12345"})
			messagePipe := core.SetupMockMessagePipe(t, ctx, pluginUnderTest)

			messagePipe.Process(core.NewMessage(test.topic, test.cmd))
			messagePipe.Run()

			assert.Eventually(t, func() bool { return len(messagePipe.GetProcessedMessages()) == len(test.msgTopics) }, 1*time.Second, 100*time.Millisecond)
			cmdr.AssertExpectations(tt)

			processedMessages := messagePipe.GetProcessedMessages()

			for idx, msg := range processedMessages {
				if test.msgTopics[idx] != msg.Topic() {
					tt.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), test.msgTopics[idx])
				}
			}

			pluginUnderTest.Close()
		})
	}
}

func testChannel(command *proto.Command) <-chan client.Message {
	c := make(chan client.Message)

	go func() {
		c <- client.MessageFromCommand(command)
	}()

	return c
}

func TestCommander_Subscriptions(t *testing.T) {
	cmdr := tutils.NewMockCommandClient()
	subs := []string{core.CommRegister, core.CommStatus, core.CommResponse, core.AgentConnected, core.Events}
	pluginUnderTest := NewCommander(cmdr, &config.Config{})

	assert.Equal(t, subs, pluginUnderTest.Subscriptions())
	cmdr.AssertExpectations(t)
}

func TestCommander_Info(t *testing.T) {
	cmdr := tutils.NewMockCommandClient()
	pluginUnderTest := NewCommander(cmdr, &config.Config{})

	assert.Equal(t, "Commander", pluginUnderTest.Info().Name())
}

func TestCommander_Close(t *testing.T) {
	cmdr := tutils.NewMockCommandClient()
	// setup expectations
	cmdr.On("Recv").Return(make(<-chan client.Message))

	pluginUnderTest := NewCommander(cmdr, &config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	messagePipe := core.SetupMockMessagePipe(t, ctx, pluginUnderTest)

	pluginUnderTest.Init(messagePipe)

	m := core.NewMessage(core.AgentConnected, &proto.Command{
		Data: &proto.Command_AgentConnectResponse{
			AgentConnectResponse: &proto.AgentConnectResponse{}},
	})

	messagePipe.Process(m)
	messagePipe.Run()
	time.Sleep(250 * time.Millisecond)

	cancel()
	pluginUnderTest.Close()

	cmdr.AssertExpectations(t)
}
