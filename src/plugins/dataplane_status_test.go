package plugins

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestDataPlaneStatus(t *testing.T) {
	tests := []struct {
		testName        string
		message         *core.Message
		expectedMessage *core.Message
	}{
		{
			testName: "default status",
			message:  nil,
			expectedMessage: core.NewMessage(core.CommStatus, &proto.Command{
				Meta: nil,
				Data: &proto.Command_DataplaneStatus{
					DataplaneStatus: &proto.DataplaneStatus{},
				},
			}),
		},
		{
			testName: "successful nginx config apply",
			message: core.NewMessage(core.NginxConfigApplySucceeded, &proto.AgentActivityStatus{
				Status: &proto.AgentActivityStatus_NginxConfigStatus{
					NginxConfigStatus: &proto.NginxConfigStatus{
						CorrelationId: "123",
						Status:        proto.NginxConfigStatus_OK,
						Message:       "config applied",
					},
				},
			}),
			expectedMessage: core.NewMessage(core.CommStatus, &proto.Command{
				Meta: nil,
				Data: &proto.Command_DataplaneStatus{
					DataplaneStatus: &proto.DataplaneStatus{
						AgentActivityStatus: []*proto.AgentActivityStatus{
							{
								Status: &proto.AgentActivityStatus_NginxConfigStatus{
									NginxConfigStatus: &proto.NginxConfigStatus{
										CorrelationId: "123",
										Status:        proto.NginxConfigStatus_OK,
										Message:       "config applied",
									},
								},
							},
						},
					},
				},
			}),
		},
		{
			testName: "nginx config apply failed",
			message: core.NewMessage(core.NginxConfigApplySucceeded, &proto.AgentActivityStatus{
				Status: &proto.AgentActivityStatus_NginxConfigStatus{
					NginxConfigStatus: &proto.NginxConfigStatus{
						CorrelationId: "123",
						Status:        proto.NginxConfigStatus_ERROR,
						Message:       "config applied failed",
					},
				},
			}),
			expectedMessage: core.NewMessage(core.CommStatus, &proto.Command{
				Meta: nil,
				Data: &proto.Command_DataplaneStatus{
					DataplaneStatus: &proto.DataplaneStatus{
						AgentActivityStatus: []*proto.AgentActivityStatus{
							{
								Status: &proto.AgentActivityStatus_NginxConfigStatus{
									NginxConfigStatus: &proto.NginxConfigStatus{
										CorrelationId: "123",
										Status:        proto.NginxConfigStatus_ERROR,
										Message:       "config applied failed",
									},
								},
							},
						},
					},
				},
			}),
		},
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

	env := tutils.NewMockEnvironment()
	env.On("Processes", mock.Anything).Return([]core.Process{})
	env.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})

	config := &config.Config{
		Server:     config.Server{},
		ConfigDirs: "",
		Log:        config.LogConfig{},
		TLS:        config.TLSConfig{},
		Dataplane: config.Dataplane{
			Status: config.Status{PollInterval: time.Duration(1)},
		},
		AgentMetrics: config.AgentMetrics{},
		Tags:         []string{},
	}

	dataPlaneStatus := NewDataPlaneStatus(config, grpc.NewMessageMeta(uuid.New().String()), binary, env, "")
	dataPlaneStatus.napDetails = testNAPDetailsActive

	messagePipe := core.NewMockMessagePipe(context.Background())
	err := messagePipe.Register(10, dataPlaneStatus)
	assert.NoError(t, err)

	messagePipe.Run()
	defer dataPlaneStatus.Close()

	for _, test := range tests {
		t.Run(test.testName, func(tt *testing.T) {
			if test.message != nil {
				messagePipe.Process(test.message)
				messagePipe.RunWithoutInit()
			}

			result := messagePipe.GetProcessedMessages()

			message := result[len(result)-1]
			assert.Equal(t, test.expectedMessage.Topic(), message.Topic())

			cmd := message.Data().(*proto.Command)
			dps := cmd.Data.(*proto.Command_DataplaneStatus)

			expectedCmd := test.expectedMessage.Data().(*proto.Command)
			expectedDps := expectedCmd.Data.(*proto.Command_DataplaneStatus)

			assert.NotNil(t, dps)
			assert.NotNil(t, dps.DataplaneStatus.GetHost().GetHostname())
			assert.Len(t, dps.DataplaneStatus.GetDataplaneSoftwareDetails(), 1)
			assert.EqualValues(t, expectedDps.DataplaneStatus.GetAgentActivityStatus(), dps.DataplaneStatus.GetAgentActivityStatus())
		})
	}
}

func TestDPSSyncAgentConfigChange(t *testing.T) {
	testCases := []struct {
		testName         string
		config           *config.Config
		expUpdatedConfig *config.Config
		updatedTags      bool
	}{
		{
			testName: "ValuesToUpdate",
			config: &config.Config{
				Tags:     tutils.InitialConfTags,
				Features: config.Defaults.Features,
			},
			expUpdatedConfig: &config.Config{
				Tags:     updateTags,
				Features: config.Defaults.Features,
			},
			updatedTags: true,
		},
		{
			testName: "NoValuesToUpdate",
			config: &config.Config{
				Tags:     tutils.InitialConfTags,
				Features: config.Defaults.Features,
			},
			expUpdatedConfig: &config.Config{
				Tags:     tutils.InitialConfTags,
				Features: config.Defaults.Features,
			},
			updatedTags: false,
		},
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

	env := tutils.NewMockEnvironment()
	env.On("Processes", mock.Anything).Return([]core.Process{})
	env.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create an agent config and initialize Viper config properties
			// based off of it, clean up when done.
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			// Setup data plane status and mock pipeline
			dataPlaneStatus := NewDataPlaneStatus(tc.config, grpc.NewMessageMeta(uuid.New().String()), binary, env, "")
			messagePipe := core.NewMockMessagePipe(context.Background())

			err = messagePipe.Register(10, dataPlaneStatus)
			assert.NoError(t, err)

			messagePipe.Run()
			defer dataPlaneStatus.Close()

			// Make sure tags are set properly before updating
			sort.Strings(tc.config.Tags)
			sort.Strings(*dataPlaneStatus.tags)
			assert.Equal(t, tc.config.Tags, *dataPlaneStatus.tags)

			// Attempt update & check results
			updated, err := config.UpdateAgentConfig("12345", tc.expUpdatedConfig.Tags, tc.expUpdatedConfig.Features)
			assert.Nil(t, err)
			assert.Equal(t, updated, tc.updatedTags)

			// Create message that should trigger a sync agent config call
			msg := core.NewMessage(core.AgentConfigChanged, "")
			dataPlaneStatus.Process(msg)

			// Check that the config was properly updated
			sort.Strings(tc.expUpdatedConfig.Tags)
			sort.Strings(*dataPlaneStatus.tags)
			assert.Equal(t, tc.expUpdatedConfig.Tags, *dataPlaneStatus.tags)
		})
	}
}

func TestDPSSyncNAPDetails(t *testing.T) {
	testCases := []struct {
		testName          string
		initialNAPDetails *proto.DataplaneSoftwareDetails_AppProtectWafDetails
		updatedNAPDetails *proto.DataplaneSoftwareDetails_AppProtectWafDetails
	}{
		{
			testName:          "NAPDetailsUpdatedSuccessfully",
			initialNAPDetails: testNAPDetailsActive,
			updatedNAPDetails: testNAPDetailsDegraded,
		},
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

	env := tutils.NewMockEnvironment()
	env.On("Processes", mock.Anything).Return([]core.Process{})
	env.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})

	config := &config.Config{
		Server:     config.Server{},
		ConfigDirs: "",
		Log:        config.LogConfig{},
		TLS:        config.TLSConfig{},
		Dataplane: config.Dataplane{
			Status: config.Status{PollInterval: time.Duration(1)},
		},
		AgentMetrics: config.AgentMetrics{},
		Tags:         []string{},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Setup DataPlaneStatus
			dataPlaneStatus := NewDataPlaneStatus(config, grpc.NewMessageMeta(uuid.New().String()), binary, env, "")
			dataPlaneStatus.napDetails = tc.initialNAPDetails
			defer dataPlaneStatus.Close()

			// Set up communication pipe and run it
			messagePipe := core.NewMockMessagePipe(context.Background())
			err := messagePipe.Register(10, dataPlaneStatus)
			assert.NoError(t, err)
			messagePipe.Run()

			// Make sure initial NAP details are as expected
			assert.Equal(t, tc.initialNAPDetails, dataPlaneStatus.napDetails)

			// Send updated NAP details message
			dataPlaneStatus.Process(core.NewMessage(core.NginxAppProtectDetailsGenerated, tc.updatedNAPDetails))

			// Check if NAP details were updated
			assert.Equal(t, tc.updatedNAPDetails, dataPlaneStatus.napDetails)
		})
	}
}

func TestDataPlaneSubscriptions(t *testing.T) {
	expectedSubscriptions := []string{
		core.AgentConfigChanged,
		core.NginxAppProtectDetailsGenerated,
		core.NginxConfigValidationPending,
		core.NginxConfigApplyFailed,
		core.NginxConfigApplySucceeded,
	}

	processID := "12345"

	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(detailsMap)
	binary.On("GetNginxIDForProcess", mock.Anything).Return(processID)
	binary.On("GetNginxDetailsFromProcess", mock.Anything).Return(detailsMap[processID])

	env := tutils.NewMockEnvironment()
	env.On("Processes", mock.Anything).Return([]core.Process{})
	env.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})

	config := &config.Config{
		Server:     config.Server{},
		ConfigDirs: "",
		Log:        config.LogConfig{},
		TLS:        config.TLSConfig{},
		Dataplane: config.Dataplane{
			Status: config.Status{PollInterval: time.Duration(1)},
		},
		AgentMetrics: config.AgentMetrics{},
		Tags:         []string{},
	}

	dataPlaneStatus := NewDataPlaneStatus(config, grpc.NewMessageMeta(uuid.New().String()), binary, env, "")

	assert.Equal(t, expectedSubscriptions, dataPlaneStatus.Subscriptions())
}
