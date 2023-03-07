/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	commonProto "github.com/nginx/agent/sdk/v2/proto/common"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestActivityEvents_Process(t *testing.T) {
	expectedCommonDimensions := []*commonProto.Dimension{
		{
			Name:  "system_id",
			Value: "12345678",
		},
		{
			Name:  "hostname",
			Value: "test-host",
		},
		{
			Name:  "instance_group",
			Value: "group-a",
		},
		{
			Name:  "system.tags",
			Value: "tag-a,tag-b",
		},
	}

	nginxDim := []*commonProto.Dimension{
		{
			Name:  "nginx_id",
			Value: "12345",
		},
	}
	expectedNginxDimensions := append(nginxDim, expectedCommonDimensions...)

	tests := []struct {
		name                string
		message             *core.Message
		msgTopics           []string
		expectedEventReport *eventsProto.EventReport
	}{
		{
			name:    "test NginxInstancesFound message",
			message: core.NewMessage(core.NginxInstancesFound, tutils.GetDetailsMap()),
			msgTopics: []string{
				core.NginxInstancesFound,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "INFO",
							Timestamp:  &types.Timestamp{Seconds: 1564894, Nanos: 894},
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-v1.2.1 master process was found with a pid 123",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test NginxReloadComplete message - reload failed",
			message: core.NewMessage(core.NginxReloadComplete, NginxReloadResponse{
				succeeded:     false,
				nginxDetails:  tutils.GetDetailsMap()["12345"],
				correlationId: uuid.NewString(),
			}),
			msgTopics: []string{
				core.NginxReloadComplete,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "ERROR",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-v1.2.1 master process (pid: 123) failed to reload",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test NginxReloadComplete message - reload succeeded",
			message: core.NewMessage(core.NginxReloadComplete, NginxReloadResponse{
				succeeded:     true,
				nginxDetails:  tutils.GetDetailsMap()["12345"],
				correlationId: uuid.NewString(),
			}),
			msgTopics: []string{
				core.NginxReloadComplete,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "WARN",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-v1.2.1 master process (pid: 123) reloaded successfully",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test successful CommResponse message apply",
			message: core.NewMessage(core.CommResponse, &proto.Command{
				Meta: grpc.NewMessageMeta(uuid.New().String()),
				Data: &proto.Command_NginxConfigResponse{
					NginxConfigResponse: &proto.NginxConfigResponse{
						Status: newOKStatus("config applied successfully").CmdStatus,
						Action: proto.NginxConfigAction_APPLY,
						ConfigData: &proto.ConfigDescriptor{
							NginxId: "12345",
						},
					},
				},
			}),
			msgTopics: []string{
				core.CommResponse,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Config",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "successfully applied config on test-host",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test successful CommResponse message force",
			message: core.NewMessage(core.CommResponse, &proto.Command{
				Meta: grpc.NewMessageMeta(uuid.New().String()),
				Data: &proto.Command_NginxConfigResponse{
					NginxConfigResponse: &proto.NginxConfigResponse{
						Status: newOKStatus("config applied successfully").CmdStatus,
						Action: proto.NginxConfigAction_FORCE,
						ConfigData: &proto.ConfigDescriptor{
							NginxId: "12345",
						},
					},
				},
			}),
			msgTopics: []string{
				core.CommResponse,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Config",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "successfully applied config on test-host",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test failed CommResponse message",
			message: core.NewMessage(core.CommResponse, &proto.Command{
				Meta: grpc.NewMessageMeta(uuid.New().String()),
				Data: &proto.Command_NginxConfigResponse{
					NginxConfigResponse: &proto.NginxConfigResponse{
						Status: newErrStatus("Config apply failed (write): some error message").CmdStatus,
						Action: proto.NginxConfigAction_APPLY,
						ConfigData: &proto.ConfigDescriptor{
							NginxId: "12345",
						},
					},
				},
			}),
			msgTopics: []string{
				core.CommResponse,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Config",
							EventLevel: "ERROR",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "failed to apply nginx config on test-host",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test CommResponse message with the NginxConfigAction not set to APPLY",
			message: core.NewMessage(core.CommResponse, &proto.Command{
				Meta: grpc.NewMessageMeta(uuid.New().String()),
				Data: &proto.Command_NginxConfigResponse{
					NginxConfigResponse: &proto.NginxConfigResponse{
						Status:     newOKStatus("config uploaded status").CmdStatus,
						Action:     proto.NginxConfigAction_RETURN,
						ConfigData: nil,
					},
				},
			}),
			msgTopics: []string{
				core.CommResponse,
			},
			expectedEventReport: nil,
		},
		{
			name: "test successful ConfigRollbackResponse message",
			message: core.NewMessage(core.ConfigRollbackResponse, ConfigRollbackResponse{
				succeeded:     true,
				nginxDetails:  tutils.GetDetailsMap()["12345"],
				correlationId: uuid.NewString(),
			}),
			msgTopics: []string{
				core.ConfigRollbackResponse,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Config",
							EventLevel: "WARN",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx config was rolled back on test-host",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name: "test failed ConfigRollbackResponse message",
			message: core.NewMessage(core.ConfigRollbackResponse, ConfigRollbackResponse{
				succeeded:     false,
				nginxDetails:  tutils.GetDetailsMap()["12345"],
				correlationId: uuid.NewString(),
			}),
			msgTopics: []string{
				core.ConfigRollbackResponse,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Config",
							EventLevel: "ERROR",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "failed to rollback nginx config on test-host",
								Dimensions: expectedNginxDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test AgentStart message",
			message: core.NewMessage(core.AgentStarted, &AgentEventMeta{version: "v0.0.1", pid: "75231"}),
			msgTopics: []string{
				core.AgentStarted,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Agent",
							Category:   "Status",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-agent v0.0.1 started on test-host with pid 75231",
								Dimensions: expectedCommonDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test NginxMasterProcCreated message",
			message: core.NewMessage(core.NginxMasterProcCreated, &proto.NginxDetails{Version: "1.0.1", ProcessId: "75231"}),
			msgTopics: []string{
				core.NginxMasterProcCreated,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-v1.0.1 master process was found with a pid 75231",
								Dimensions: expectedCommonDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test NginxMasterProcKilled message",
			message: core.NewMessage(core.NginxMasterProcKilled, &proto.NginxDetails{Version: "1.0.1", ProcessId: "75231"}),
			msgTopics: []string{
				core.NginxMasterProcKilled,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "WARN",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "nginx-v1.0.1 master process (pid: 75231) stopped",
								Dimensions: expectedCommonDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test NginxWorkerProcCreated message",
			message: core.NewMessage(core.NginxWorkerProcCreated, &proto.NginxDetails{Version: "1.0.1", ProcessId: "75231"}),
			msgTopics: []string{
				core.NginxWorkerProcCreated,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "new worker process started with pid 75231 for nginx-v1.0.1 process (pid: 75231)",
								Dimensions: expectedCommonDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test NginxWorkerProcKilled message",
			message: core.NewMessage(core.NginxWorkerProcKilled, &proto.NginxDetails{Version: "1.0.1", ProcessId: "75231"}),
			msgTopics: []string{
				core.NginxWorkerProcKilled,
				core.Events,
			},
			expectedEventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{
					{
						Metadata: &eventsProto.Metadata{
							Module:     "NGINX-AGENT",
							Type:       "Nginx",
							Category:   "Status",
							EventLevel: "INFO",
						},
						Data: &eventsProto.Event_ActivityEvent{
							ActivityEvent: &eventsProto.ActivityEvent{
								Message:    "worker process with pid 75231 is shutting down for nginx-v1.0.1 process (pid: 75231)",
								Dimensions: expectedCommonDimensions,
							},
						},
					},
				},
			},
		},
		{
			name:    "test unknown message",
			message: core.NewMessage(core.UNKNOWN, "unknown message"),
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

			ctx, cancelCTX := context.WithCancel(context.Background())
			env := tutils.NewMockEnvironment()
			config := &config.Config{
				ClientID:      "12345",
				InstanceGroup: "group-a",
				Tags: []string{
					"tag-a",
					"tag-b",
				},
			}

			pluginUnderTest := NewEvents(config, env, grpc.NewMessageMeta(uuid.New().String()), core.NewNginxBinary(env, config))

			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			messagePipe.Process(test.message)
			messagePipe.Run()
			time.Sleep(250 * time.Millisecond)

			processedMessages := messagePipe.GetProcessedMessages()
			if len(processedMessages) != len(test.msgTopics) {
				tt.Fatalf("expected %d messages, received %d", len(test.msgTopics), len(processedMessages))
			}
			for idx, msg := range processedMessages {
				if test.msgTopics[idx] != msg.Topic() {
					tt.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), test.msgTopics[idx])
				}
				if test.expectedEventReport != nil && msg.Exact(core.Events) {
					expectedEvent := test.expectedEventReport.Events[0]
					actualEvent := msg.Data().(*proto.Command).GetEventReport().Events[0]

					// assert metadata
					assert.Equal(tt, expectedEvent.Metadata.Module, actualEvent.Metadata.Module)
					assert.Equal(tt, expectedEvent.Metadata.Category, actualEvent.Metadata.Category)
					assert.Equal(tt, expectedEvent.Metadata.Type, actualEvent.Metadata.Type)
					assert.Equal(tt, expectedEvent.Metadata.EventLevel, actualEvent.Metadata.EventLevel)

					// only assert timestamp when we can predict it
					if expectedEvent.Metadata.Timestamp != nil {
						assert.Equal(tt, expectedEvent.Metadata.Timestamp, actualEvent.Metadata.Timestamp)
					}

					// assert activity event
					assert.Equal(tt, expectedEvent.GetActivityEvent().Message, actualEvent.GetActivityEvent().Message)
					assert.Equal(tt, expectedEvent.GetActivityEvent().Dimensions, actualEvent.GetActivityEvent().Dimensions)
				}
			}

			cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestGenerateAgentStopEvent(t *testing.T) {
	expectedCommonDimensions := []*commonProto.Dimension{
		{
			Name:  "system_id",
			Value: "12345678",
		},
		{
			Name:  "hostname",
			Value: "test-host",
		},
		{
			Name:  "instance_group",
			Value: "group-a",
		},
		{
			Name:  "system.tags",
			Value: "tag-a,tag-b",
		},
	}

	tests := []struct {
		name          string
		agentVersion  string
		pid           string
		conf          *config.Config
		expectedEvent *eventsProto.Event
	}{
		{
			name:         "test AgentStop message",
			agentVersion: "v0.0.1",
			pid:          "212121",
			conf: &config.Config{
				ClientID:      "12345",
				InstanceGroup: "group-a",
				Tags: []string{
					"tag-a",
					"tag-b",
				},
			},
			expectedEvent: &eventsProto.Event{
				Metadata: &eventsProto.Metadata{
					Module:     "NGINX-AGENT",
					Type:       "Agent",
					Category:   "Status",
					EventLevel: "WARN",
				},
				Data: &eventsProto.Event_ActivityEvent{
					ActivityEvent: &eventsProto.ActivityEvent{
						Message:    "nginx-agent v0.0.1 (pid: 212121) stopped on test-host",
						Dimensions: expectedCommonDimensions,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tutils.NewMockEnvironment()

			agentStopCmd := GenerateAgentStopEventCommand(&AgentEventMeta{version: tt.agentVersion, pid: tt.pid}, tt.conf, env)
			actualEvent := agentStopCmd.GetEventReport().Events[0]

			// assert metadata
			assert.Equal(t, tt.expectedEvent.Metadata.Module, actualEvent.Metadata.Module)
			assert.Equal(t, tt.expectedEvent.Metadata.Category, actualEvent.Metadata.Category)
			assert.Equal(t, tt.expectedEvent.Metadata.Type, actualEvent.Metadata.Type)
			assert.Equal(t, tt.expectedEvent.Metadata.EventLevel, actualEvent.Metadata.EventLevel)

			// assert activity event
			assert.Equal(t, tt.expectedEvent.GetActivityEvent().Message, actualEvent.GetActivityEvent().Message)
			assert.Equal(t, tt.expectedEvent.GetActivityEvent().Dimensions, actualEvent.GetActivityEvent().Dimensions)
		})
	}
}
