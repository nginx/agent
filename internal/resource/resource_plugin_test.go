// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/nginx/agent/v3/test/stub"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginxinc/nginx-plus-go-client/v2/client"

	"github.com/nginx/agent/v3/internal/bus/busfakes"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/test/types"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/resource/resourcefakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Process(t *testing.T) {
	ctx := context.Background()

	updatedInstance := &mpi.Instance{
		InstanceConfig: protos.NginxOssInstance([]string{}).GetInstanceConfig(),
		InstanceMeta:   protos.NginxOssInstance([]string{}).GetInstanceMeta(),
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  56789,
			BinaryPath: protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetBinaryPath(),
			ConfigPath: protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetConfigPath(),
			Details:    protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetDetails(),
		},
	}

	tests := []struct {
		name     string
		message  *bus.Message
		resource *mpi.Resource
		topic    string
	}{
		{
			name: "Test 1: New Instance Topic",
			message: &bus.Message{
				Topic: bus.AddInstancesTopic,
				Data: []*mpi.Instance{
					protos.NginxOssInstance([]string{}),
				},
			},
			resource: protos.HostResource(),
			topic:    bus.ResourceUpdateTopic,
		},
		{
			name: "Test 2: Update Instance Topic",
			message: &bus.Message{
				Topic: bus.UpdatedInstancesTopic,
				Data: []*mpi.Instance{
					updatedInstance,
				},
			},
			resource: &mpi.Resource{
				ResourceId: protos.HostResource().GetResourceId(),
				Instances: []*mpi.Instance{
					updatedInstance,
				},
				Info: protos.HostResource().GetInfo(),
			},
			topic: bus.ResourceUpdateTopic,
		},
		{
			name: "Test 3: Delete Instance Topic",
			message: &bus.Message{
				Topic: bus.DeletedInstancesTopic,
				Data: []*mpi.Instance{
					updatedInstance,
				},
			},
			resource: &mpi.Resource{
				ResourceId: protos.HostResource().GetResourceId(),
				Instances:  []*mpi.Instance{},
				Info:       protos.HostResource().GetInfo(),
			},
			topic: bus.ResourceUpdateTopic,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.AddInstancesReturns(protos.HostResource())
			fakeResourceService.UpdateInstancesReturns(test.resource)
			fakeResourceService.DeleteInstancesReturns(test.resource)
			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic, messagePipe.Messages()[0].Topic)
			assert.Equal(t, test.resource, messagePipe.Messages()[0].Data)
		})
	}
}

func TestResource_Process_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		message  *bus.Message
		applyErr error
		topic    []string
	}{
		{
			name: "Test 1: Write Config Successful Topic - Success Status",
			message: &bus.Message{
				Topic: bus.WriteConfigSuccessfulTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         nil,
				},
			},
			applyErr: nil,
			topic:    []string{bus.ConfigApplySuccessfulTopic},
		},
		{
			name: "Test 2: Write Config Successful Topic - Fail Status",
			message: &bus.Message{
				Topic: bus.WriteConfigSuccessfulTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         nil,
				},
			},
			applyErr: errors.New("error reloading"),
			topic:    []string{bus.DataPlaneResponseTopic, bus.ConfigApplyFailedTopic},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.ApplyConfigReturns(&model.NginxConfigContext{}, test.applyErr)
			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic[0], messagePipe.Messages()[0].Topic)

			if len(test.topic) > 1 {
				assert.Equal(t, test.topic[1], messagePipe.Messages()[1].Topic)
			}

			if test.applyErr != nil {
				response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
				assert.True(tt, ok)
				assert.Equal(tt, test.applyErr.Error(), response.GetCommandResponse().GetError())
			}
		})
	}
}

func TestResource_createPlusAPIError(t *testing.T) {
	s := "failed to get the HTTP servers of upstream nginx1: expected 200 response, got 404. error.status=404;" +
		" error.text=upstream not found; error.code=UpstreamNotFound; request_id=b534bdab5cb5e321e8b41b431828b270; " +
		"href=https://nginx.org/en/docs/http/ngx_http_api_module.html"

	expectedErr := plusAPIErr{
		Error: errResponse{
			Status: "404",
			Text:   "upstream not found",
			Code:   "UpstreamNotFound",
		},
		RequestID: "b534bdab5cb5e321e8b41b431828b270",
		Href:      "https://nginx.org/en/docs/http/ngx_http_api_module.html",
	}
	expectedJSON, err := json.Marshal(expectedErr)
	require.NoError(t, err)

	result := createPlusAPIError(errors.New(s))

	assert.Equal(t, errors.New(string(expectedJSON)), result)
}

// nolint: dupl
func TestResource_Process_APIAction_GetHTTPServers(t *testing.T) {
	ctx := context.Background()

	inValidInstance := protos.NginxPlusInstance([]string{})
	inValidInstance.InstanceMeta.InstanceId = "e1374cb1-462d-3b6c-9f3b-f28332b5f10f"

	tests := []struct {
		instance  *mpi.Instance
		name      string
		message   *bus.Message
		err       error
		topic     []string
		upstreams []client.UpstreamServer
	}{
		{
			name: "Test 1: Success, Get HTTP Server API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: nil,
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},

		{
			name: "Test 2: Fail, Get HTTP Server API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to get http servers"),
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 3: Fail, OSS Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, instance is not NGINX Plus"),
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxOssInstance([]string{}),
		},
		{
			name: "Test 4: Fail, No Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, could not find instance with ID: " +
				"e1374cb1-462d-3b6c-9f3b-f28332b5f10c"),
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: inValidInstance,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.GetHTTPUpstreamServersReturns(test.upstreams, test.err)
			if test.instance.GetInstanceMeta().GetInstanceId() != "e1374cb1-462d-3b6c-9f3b-f28332b5f10f" {
				fakeResourceService.InstanceReturns(test.instance)
			}

			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic[0], messagePipe.Messages()[0].Topic)

			response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
			assert.True(tt, ok)

			if test.err != nil {
				assert.Equal(tt, test.err.Error(), response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
			} else {
				assert.Empty(t, response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
			}
		})
	}
}

// nolint: dupl
func TestResource_Process_APIAction_UpdateHTTPUpstreams(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		instance    *mpi.Instance
		name        string
		expectedLog string
		message     *bus.Message
		err         error
		topic       []string
		upstreams   []client.UpstreamServer
	}{
		{
			name: "Test 1: Success, Update HTTP Upstream Servers",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusUpdateHTTPServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(), []*structpb.Struct{
						{
							Fields: map[string]*structpb.Value{
								"max_cons":  structpb.NewNumberValue(8),
								"max_fails": structpb.NewNumberValue(0),
								"backup":    structpb.NewBoolValue(true),
								"service":   structpb.NewStringValue("test_server"),
							},
						},
					}),
			},
			err: nil,
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:       []string{bus.DataPlaneResponseTopic},
			instance:    protos.NginxPlusInstance([]string{}),
			expectedLog: "Successfully updated http upstream",
		},
		{
			name: "Test 2: Fail, Update HTTP Upstream Servers",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusUpdateHTTPServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(), []*structpb.Struct{
						{
							Fields: map[string]*structpb.Value{
								"max_cons":  structpb.NewNumberValue(8),
								"max_fails": structpb.NewNumberValue(0),
								"backup":    structpb.NewBoolValue(true),
								"service":   structpb.NewStringValue("test_server"),
							},
						},
					}),
			},
			err: errors.New("something went wrong"),
			upstreams: []client.UpstreamServer{
				helpers.CreateNginxPlusUpstreamServer(t),
			},
			topic:       []string{bus.DataPlaneResponseTopic},
			instance:    protos.NginxPlusInstance([]string{}),
			expectedLog: "Unable to update HTTP servers of upstream",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.InstanceReturns(test.instance)
			fakeResourceService.UpdateHTTPUpstreamServersReturnsOnCall(0, test.upstreams, []client.UpstreamServer{},
				[]client.UpstreamServer{}, test.err)
			fakeResourceService.UpdateHTTPUpstreamServersReturnsOnCall(1, []client.UpstreamServer{},
				[]client.UpstreamServer{}, []client.UpstreamServer{}, test.err)

			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(tt, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(tt, test.topic[0], messagePipe.Messages()[0].Topic)

			response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
			assert.True(tt, ok)

			if test.err != nil {
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
			} else {
				assert.Empty(tt, response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
			}

			helpers.ValidateLog(tt, test.expectedLog, logBuf)
		})
	}
}

// nolint: dupl
func TestResource_Process_APIAction_UpdateStreamServers(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		instance    *mpi.Instance
		name        string
		expectedLog string
		message     *bus.Message
		err         error
		topic       []string
		upstreams   []client.StreamUpstreamServer
	}{
		{
			name: "Test 1: Success, Update Stream Servers",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusUpdateStreamServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(), []*structpb.Struct{
						{
							Fields: map[string]*structpb.Value{
								"max_cons":  structpb.NewNumberValue(8),
								"max_fails": structpb.NewNumberValue(0),
								"backup":    structpb.NewBoolValue(true),
								"service":   structpb.NewStringValue("test_server"),
							},
						},
					}),
			},
			err: nil,
			upstreams: []client.StreamUpstreamServer{
				helpers.CreateNginxPlusStreamServer(t),
			},
			topic:       []string{bus.DataPlaneResponseTopic},
			instance:    protos.NginxPlusInstance([]string{}),
			expectedLog: "Successfully updated stream upstream",
		},
		{
			name: "Test 2: Fail, Update Stream Servers",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusUpdateStreamServers("test_upstream",
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(), []*structpb.Struct{
						{
							Fields: map[string]*structpb.Value{
								"max_cons":  structpb.NewNumberValue(8),
								"max_fails": structpb.NewNumberValue(0),
								"backup":    structpb.NewBoolValue(true),
								"service":   structpb.NewStringValue("test_server"),
							},
						},
					}),
			},
			err: errors.New("something went wrong"),
			upstreams: []client.StreamUpstreamServer{
				helpers.CreateNginxPlusStreamServer(t),
			},
			topic:       []string{bus.DataPlaneResponseTopic},
			instance:    protos.NginxPlusInstance([]string{}),
			expectedLog: "Unable to update stream servers of upstream",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.InstanceReturns(test.instance)
			fakeResourceService.UpdateStreamServersReturnsOnCall(0, test.upstreams, []client.StreamUpstreamServer{},
				[]client.StreamUpstreamServer{}, test.err)
			fakeResourceService.UpdateStreamServersReturnsOnCall(0, test.upstreams, []client.StreamUpstreamServer{},
				[]client.StreamUpstreamServer{}, test.err)

			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(tt, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(tt, test.topic[0], messagePipe.Messages()[0].Topic)

			response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
			assert.True(tt, ok)

			if test.err != nil {
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
			} else {
				assert.Empty(tt, response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
			}

			helpers.ValidateLog(tt, test.expectedLog, logBuf)
		})
	}
}

// nolint: dupl
func TestResource_Process_APIAction_GetStreamUpstreams(t *testing.T) {
	ctx := context.Background()

	inValidInstance := protos.NginxPlusInstance([]string{})
	inValidInstance.InstanceMeta.InstanceId = "e1374cb1-462d-3b6c-9f3b-f28332b5f10f"

	tests := []struct {
		instance  *mpi.Instance
		upstreams *client.StreamUpstreams
		name      string
		message   *bus.Message
		err       error
		topic     []string
	}{
		{
			name: "Test 1: Success, Get Stream Upstreams API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusGetStreamUpstreams(
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: nil,
			upstreams: &client.StreamUpstreams{
				"upstream_1": client.StreamUpstream{
					Zone: "zone_1",
					Peers: []client.StreamPeer{
						{
							Server: "server_1",
						},
					},
					Zombies: 0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 2: Fail, Get Stream Upstreams API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusGetStreamUpstreams(
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to get stream upstreams servers"),
			upstreams: &client.StreamUpstreams{
				"upstream_1": client.StreamUpstream{
					Zone: "zone_1",
					Peers: []client.StreamPeer{
						{
							Server: "server_1",
						},
					},
					Zombies: 0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 3: Fail, No Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, could not find instance with ID: " +
				"e1374cb1-462d-3b6c-9f3b-f28332b5f10c"),
			upstreams: &client.StreamUpstreams{
				"upstream_1": client.StreamUpstream{
					Zone: "zone_1",
					Peers: []client.StreamPeer{
						{
							Server: "server_1",
						},
					},
					Zombies: 0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: inValidInstance,
		},
		{
			name: "Test 4: Fail, OSS Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, instance is not NGINX Plus"),
			upstreams: &client.StreamUpstreams{
				"upstream_1": client.StreamUpstream{
					Zone: "zone_1",
					Peers: []client.StreamPeer{
						{
							Server: "server_1",
						},
					},
					Zombies: 0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxOssInstance([]string{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.GetStreamUpstreamsReturns(test.upstreams, test.err)
			if test.instance.GetInstanceMeta().GetInstanceId() != "e1374cb1-462d-3b6c-9f3b-f28332b5f10f" {
				fakeResourceService.InstanceReturns(test.instance)
			}

			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic[0], messagePipe.Messages()[0].Topic)

			response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
			assert.True(tt, ok)

			if test.err != nil {
				assert.Equal(tt, test.err.Error(), response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
			} else {
				assert.Empty(t, response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
			}
		})
	}
}

// nolint: dupl
func TestResource_Process_APIAction_GetUpstreams(t *testing.T) {
	ctx := context.Background()

	inValidInstance := protos.NginxPlusInstance([]string{})
	inValidInstance.InstanceMeta.InstanceId = "e1374cb1-462d-3b6c-9f3b-f28332b5f10f"

	tests := []struct {
		instance  *mpi.Instance
		upstreams *client.Upstreams
		name      string
		message   *bus.Message
		err       error
		topic     []string
	}{
		{
			name: "Test 1: Success, Get Upstreams API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusGetUpstreams(
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: nil,
			upstreams: &client.Upstreams{
				"upstream_1": client.Upstream{
					Zone: "zone_1",
					Peers: []client.Peer{
						{
							Server: "server_1",
						},
					},
					Queue:     client.Queue{},
					Keepalive: 6,
					Zombies:   0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 2: Fail, Get Upstreams API Action",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreateAPIActionRequestNginxPlusGetUpstreams(
					protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to get upstreams"),
			upstreams: &client.Upstreams{
				"upstream_1": client.Upstream{
					Zone: "zone_1",
					Peers: []client.Peer{
						{
							Server: "server_1",
						},
					},
					Queue:     client.Queue{},
					Keepalive: 6,
					Zombies:   0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 3: Fail, No Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, could not find instance with ID: " +
				"e1374cb1-462d-3b6c-9f3b-f28332b5f10c"),
			upstreams: &client.Upstreams{
				"upstream_1": client.Upstream{
					Zone: "zone_1",
					Peers: []client.Peer{
						{
							Server: "server_1",
						},
					},
					Queue:     client.Queue{},
					Keepalive: 6,
					Zombies:   0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: inValidInstance,
		},
		{
			name: "Test 4: Fail, OSS Instance",
			message: &bus.Message{
				Topic: bus.APIActionRequestTopic,
				Data: protos.CreatAPIActionRequestNginxPlusGetHTTPServers("test_upstream",
					protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()),
			},
			err: errors.New("failed to preform API action, instance is not NGINX Plus"),
			upstreams: &client.Upstreams{
				"upstream_1": client.Upstream{
					Zone: "zone_1",
					Peers: []client.Peer{
						{
							Server: "server_1",
						},
					},
					Queue:     client.Queue{},
					Keepalive: 6,
					Zombies:   0,
				},
			},
			topic:    []string{bus.DataPlaneResponseTopic},
			instance: protos.NginxOssInstance([]string{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.GetUpstreamsReturns(test.upstreams, test.err)
			if test.instance.GetInstanceMeta().GetInstanceId() != "e1374cb1-462d-3b6c-9f3b-f28332b5f10f" {
				fakeResourceService.InstanceReturns(test.instance)
			}

			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic[0], messagePipe.Messages()[0].Topic)

			response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
			assert.True(tt, ok)

			if test.err != nil {
				assert.Equal(tt, test.err.Error(), response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
			} else {
				assert.Empty(t, response.GetCommandResponse().GetError())
				assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
			}
		})
	}
}

func TestResource_Process_Rollback(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		message     *bus.Message
		rollbackErr error
		topic       []string
	}{
		{
			name: "Test 1: Rollback Write Topic - Success Status",
			message: &bus.Message{
				Topic: bus.RollbackWriteTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         errors.New("something went wrong with config apply"),
				},
			},
			rollbackErr: nil,
			topic:       []string{bus.ConfigApplyCompleteTopic},
		},
		{
			name: "Test 2: Rollback Write Topic - Fail Status",
			message: &bus.Message{
				Topic: bus.RollbackWriteTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "",
					InstanceID:    protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         errors.New("something went wrong with config apply"),
				},
			},
			rollbackErr: errors.New("error reloading"),
			topic:       []string{bus.ConfigApplyCompleteTopic, bus.DataPlaneResponseTopic},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.ApplyConfigReturns(&model.NginxConfigContext{}, test.rollbackErr)
			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			sort.Slice(messagePipe.Messages(), func(i, j int) bool {
				return messagePipe.Messages()[i].Topic < messagePipe.Messages()[j].Topic
			})

			assert.Len(tt, messagePipe.Messages(), len(test.topic))

			assert.Equal(t, test.topic[0], messagePipe.Messages()[0].Topic)

			if len(test.topic) > 1 {
				assert.Equal(t, test.topic[1], messagePipe.Messages()[1].Topic)
			}

			if test.rollbackErr != nil {
				rollbackResponse, ok := messagePipe.Messages()[1].Data.(*mpi.DataPlaneResponse)
				assert.True(tt, ok)
				assert.Equal(t, test.topic[1], messagePipe.Messages()[1].Topic)
				assert.Equal(tt, test.rollbackErr.Error(), rollbackResponse.GetCommandResponse().GetError())
			}
		})
	}
}

func TestResource_Subscriptions(t *testing.T) {
	resourcePlugin := NewResource(types.AgentConfig())
	assert.Equal(t,
		[]string{
			bus.AddInstancesTopic,
			bus.UpdatedInstancesTopic,
			bus.DeletedInstancesTopic,
			bus.WriteConfigSuccessfulTopic,
			bus.RollbackWriteTopic,
			bus.APIActionRequestTopic,
		},
		resourcePlugin.Subscriptions())
}

func TestResource_Info(t *testing.T) {
	resourcePlugin := NewResource(types.AgentConfig())
	assert.Equal(t, &bus.Info{Name: "resource"}, resourcePlugin.Info())
}

func TestResource_Init(t *testing.T) {
	ctx := context.Background()
	resourceService := resourcefakes.FakeResourceServiceInterface{}

	messagePipe := busfakes.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourcePlugin := NewResource(types.AgentConfig())
	resourcePlugin.resourceService = &resourceService
	err := resourcePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.Messages()

	assert.Empty(t, messages)
}
