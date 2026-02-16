// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/file/filefakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/stub"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/nginx-plus-go-client/v3/client"

	"github.com/nginx/agent/v3/internal/bus/busfakes"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/test/types"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/nginx/nginxfakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNginx_createPlusAPIError(t *testing.T) {
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

func TestNginx_Process_APIAction_GetHTTPServers(t *testing.T) {
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
		runNginxTestHelper(t, ctx, test.name, func(fakeService *nginxfakes.FakeNginxServiceInterface) {
			fakeService.GetHTTPUpstreamServersReturns(test.upstreams, test.err)
		}, test.instance, test.message, test.topic, test.err)
	}
}

//nolint:dupl // need to refactor so that redundant code can be removed
func TestNginx_Process_APIAction_UpdateHTTPUpstreams(t *testing.T) {
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

			fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
			fakeNginxService.InstanceReturns(test.instance)
			fakeNginxService.UpdateHTTPUpstreamServersReturnsOnCall(0, test.upstreams, []client.UpstreamServer{},
				[]client.UpstreamServer{}, test.err)
			fakeNginxService.UpdateHTTPUpstreamServersReturnsOnCall(1, []client.UpstreamServer{},
				[]client.UpstreamServer{}, []client.UpstreamServer{}, test.err)

			messagePipe := busfakes.NewFakeMessagePipe()

			fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
			nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
			nginxPlugin.nginxService = fakeNginxService

			err := messagePipe.Register(2, []bus.Plugin{nginxPlugin})
			require.NoError(tt, err)

			nginxPlugin.messagePipe = messagePipe

			nginxPlugin.Process(ctx, test.message)

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

//nolint:dupl // need to refactor so that redundant code can be removed
func TestNginx_Process_APIAction_UpdateStreamServers(t *testing.T) {
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

			fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
			fakeNginxService.InstanceReturns(test.instance)
			fakeNginxService.UpdateStreamServersReturnsOnCall(0, test.upstreams, []client.StreamUpstreamServer{},
				[]client.StreamUpstreamServer{}, test.err)
			fakeNginxService.UpdateStreamServersReturnsOnCall(0, test.upstreams, []client.StreamUpstreamServer{},
				[]client.StreamUpstreamServer{}, test.err)

			messagePipe := busfakes.NewFakeMessagePipe()

			fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
			nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
			nginxPlugin.nginxService = fakeNginxService

			err := messagePipe.Register(2, []bus.Plugin{nginxPlugin})
			require.NoError(tt, err)

			nginxPlugin.messagePipe = messagePipe

			nginxPlugin.Process(ctx, test.message)

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

func TestNginx_Process_APIAction_GetStreamUpstreams(t *testing.T) {
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
			fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
			fakeNginxService.GetStreamUpstreamsReturns(test.upstreams, test.err)
			if test.instance.GetInstanceMeta().GetInstanceId() != "e1374cb1-462d-3b6c-9f3b-f28332b5f10f" {
				fakeNginxService.InstanceReturns(test.instance)
			}

			messagePipe := busfakes.NewFakeMessagePipe()

			fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
			nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
			nginxPlugin.nginxService = fakeNginxService

			err := messagePipe.Register(2, []bus.Plugin{nginxPlugin})
			require.NoError(t, err)

			nginxPlugin.messagePipe = messagePipe

			nginxPlugin.Process(ctx, test.message)

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

func TestNginx_Process_APIAction_GetUpstreams(t *testing.T) {
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
		runNginxTestHelper(t, ctx, test.name, func(fakeService *nginxfakes.FakeNginxServiceInterface) {
			fakeService.GetUpstreamsReturns(test.upstreams, test.err)
		}, test.instance, test.message, test.topic, test.err)
	}
}

func TestNginx_Subscriptions(t *testing.T) {
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	assert.Equal(t,
		[]string{
			bus.APIActionRequestTopic,
			bus.ConnectionResetTopic,
			bus.ConnectionCreatedTopic,
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
			bus.ResourceUpdateTopic,
			bus.ConfigApplyRequestTopic,
		},
		nginxPlugin.Subscriptions())

	readNginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Auxiliary, &sync.RWMutex{})
	assert.Equal(t,
		[]string{
			bus.APIActionRequestTopic,
			bus.ConnectionResetTopic,
			bus.ConnectionCreatedTopic,
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
			bus.ResourceUpdateTopic,
		},
		readNginxPlugin.Subscriptions())
}

func TestNginx_Info(t *testing.T) {
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	assert.Equal(t, &bus.Info{Name: "nginx"}, nginxPlugin.Info())

	readNginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Auxiliary, &sync.RWMutex{})
	assert.Equal(t, &bus.Info{Name: "auxiliary-nginx"}, readNginxPlugin.Info())
}

func TestNginx_Init(t *testing.T) {
	ctx := context.Background()
	fakeNginxService := nginxfakes.FakeNginxServiceInterface{}

	messagePipe := busfakes.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	nginxPlugin.nginxService = &fakeNginxService
	err := nginxPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.Messages()

	assert.Empty(t, messages)
}

func TestNginx_Process_handleConfigUploadRequest(t *testing.T) {
	ctx := context.Background()

	tempDir := os.TempDir()
	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	fileMeta := protos.FileMeta(testFile.Name(), "")

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{
						{
							FileMeta: fileMeta,
						},
						{
							FileMeta: fileMeta,
						},
					},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "123",
						Version:    "f33ref3d32d3c32d3a",
					},
				},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	err := nginxPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateFileCallCount() == 2 },
		2*time.Second,
		10*time.Millisecond,
	)

	messages := messagePipe.Messages()
	assert.Len(t, messages, 1)
	assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)

	dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_OK,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)
}

func TestNginx_Process_handleConfigUploadRequest_Failure(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/unknown/file.conf", "")

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{
						{
							FileMeta: fileMeta,
						},
						{
							FileMeta: fileMeta,
						},
					},
					ConfigVersion: protos.CreateConfigVersion(),
				},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	err := nginxPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.Messages()) == 1 },
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, 0, fakeFileServiceClient.UpdateFileCallCount())

	messages := messagePipe.Messages()
	assert.Len(t, messages, 1)

	assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)

	dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)
}

func TestNginx_Process_handleConfigApplyRequest(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	filePath := tempDir + "/nginx.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: protos.CreateConfigApplyRequest(protos.FileOverview(filePath, fileHash)),
		},
	}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	tests := []struct {
		message               *mpi.ManagementPlaneRequest
		configApplyReturnsErr error
		name                  string
		configApplyStatus     model.WriteStatus
	}{
		{
			name:                  "Test 1 - Success",
			configApplyReturnsErr: nil,
			configApplyStatus:     model.OK,
			message:               message,
		},
		{
			name:                  "Test 2 - Fail, Rollback",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.RollbackRequired,
			message:               message,
		},
		{
			name:                  "Test 3 - Fail, No Rollback",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.Error,
			message:               message,
		},
		{
			name:                  "Test 4 - Fail to cast payload",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.Error,
			message:               nil,
		},
		{
			name:                  "Test 5 - No changes needed",
			configApplyReturnsErr: nil,
			configApplyStatus:     model.NoChange,
			message:               message,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
			fakeNginxService.ApplyConfigReturns(&model.NginxConfigContext{}, test.configApplyReturnsErr)

			fakeFileManagerService := &filefakes.FakeFileManagerServiceInterface{}
			fakeFileManagerService.ConfigApplyReturns(test.configApplyStatus, test.configApplyReturnsErr)
			messagePipe := busfakes.NewFakeMessagePipe()

			nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})

			err := nginxPlugin.Init(ctx, messagePipe)
			nginxPlugin.fileManagerService = fakeFileManagerService
			nginxPlugin.nginxService = fakeNginxService
			require.NoError(t, err)

			nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyRequestTopic, Data: test.message})

			messages := messagePipe.Messages()

			switch {
			case test.configApplyStatus == model.OK:
				assert.Len(t, messages, 2)
				assert.Equal(t, bus.EnableWatchersTopic, messages[0].Topic)
				assert.Equal(t, bus.DataPlaneResponseTopic, messages[1].Topic)

				msg, ok := messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, msg.GetCommandResponse().GetStatus())
				assert.Equal(t, "Config apply successful", msg.GetCommandResponse().GetMessage())
			case test.configApplyStatus == model.RollbackRequired:
				assert.Len(t, messages, 3)

				assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_ERROR,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
				assert.Equal(t, "Config apply failed, rolling back config",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, test.configApplyReturnsErr.Error(), dataPlaneResponse.GetCommandResponse().GetError())

				dataPlaneResponse, ok = messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, "Config apply failed, rollback successful",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus())

				assert.Equal(t, bus.EnableWatchersTopic, messages[2].Topic)
			case test.configApplyStatus == model.NoChange:
				assert.Len(t, messages, 2)

				response, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, 1, fakeFileManagerService.ClearCacheCallCount())
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_OK,
					response.GetCommandResponse().GetStatus(),
				)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_OK,
					response.GetCommandResponse().GetStatus(),
				)

				assert.Equal(t, bus.EnableWatchersTopic, messages[1].Topic)

			case test.message == nil:
				assert.Empty(t, messages)
			default:
				assert.Len(t, messages, 2)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
				assert.Equal(t, "Config apply failed", dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, test.configApplyReturnsErr.Error(), dataPlaneResponse.GetCommandResponse().GetError())
				assert.Equal(t, bus.EnableWatchersTopic, messages[1].Topic)
			}
		})
	}
}

func TestNginxPlugin_Failed_ConfigApply(t *testing.T) {
	ctx := context.Background()

	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}

	tests := []struct {
		rollbackError      error
		rollbackWriteError error
		message            string
		name               string
	}{
		{
			name:               "Test 1 - Rollback Success",
			message:            "",
			rollbackError:      nil,
			rollbackWriteError: nil,
		},
		{
			name:               "Test 2 - Rollback Failed",
			message:            "config apply error: something went wrong\nrollback error: rollback failed",
			rollbackError:      errors.New("rollback failed"),
			rollbackWriteError: nil,
		},
		{
			name:               "Test 3 - Rollback Write Failed",
			message:            "config apply error: something went wrong\nrollback error: rollback write failed",
			rollbackError:      nil,
			rollbackWriteError: errors.New("rollback write failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
			fakeNginxService.ApplyConfigReturnsOnCall(0, &model.NginxConfigContext{},
				errors.New("something went wrong"))
			fakeNginxService.ApplyConfigReturnsOnCall(1, &model.NginxConfigContext{}, tt.rollbackWriteError)

			fakeFileManagerService := &filefakes.FakeFileManagerServiceInterface{}
			fakeFileManagerService.RollbackReturns(tt.rollbackError)

			messagePipe := busfakes.NewFakeMessagePipe()

			nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})

			err := nginxPlugin.Init(ctx, messagePipe)
			nginxPlugin.fileManagerService = fakeFileManagerService
			nginxPlugin.nginxService = fakeNginxService
			require.NoError(t, err)

			nginxPlugin.applyConfig(ctx, "dfsbhj6-bc92-30c1-a9c9-85591422068e", protos.
				NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId())

			messages := messagePipe.Messages()

			dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
			assert.True(t, ok)
			assert.Equal(
				t,
				mpi.CommandResponse_COMMAND_STATUS_ERROR,
				dataPlaneResponse.GetCommandResponse().GetStatus(),
			)
			assert.Equal(t, "Config apply failed, rolling back config",
				dataPlaneResponse.GetCommandResponse().GetMessage())

			if tt.rollbackError == nil && tt.rollbackWriteError == nil {
				assert.Len(t, messages, 3)
				dataPlaneResponse, ok = messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)

				assert.Equal(t, "Config apply failed, rollback successful",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, bus.EnableWatchersTopic, messages[2].Topic)
			} else {
				assert.Len(t, messages, 4)
				dataPlaneResponse, ok = messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_ERROR,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)

				assert.Equal(t, "Rollback failed", dataPlaneResponse.GetCommandResponse().GetMessage())

				dataPlaneResponse, ok = messages[2].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)

				assert.Equal(t, "Config apply failed, rollback failed",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, tt.message, dataPlaneResponse.GetCommandResponse().GetError())
				assert.Equal(t, bus.EnableWatchersTopic, messages[3].Topic)
			}
		})
	}
}

func TestNginxPlugin_Process_NginxConfigUpdateTopic(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/etc/nginx/nginx/conf", "")

	message := &model.NginxConfigContext{
		Files: []*mpi.File{
			{
				FileMeta: fileMeta,
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.UpdateOverviewReturns(&mpi.UpdateOverviewResponse{
		Overview: nil,
	}, nil)

	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	err := nginxPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	nginxPlugin.Process(ctx, &bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateOverviewCallCount() == 1 },
		2*time.Second,
		10*time.Millisecond,
	)
}

//nolint:revive,lll // maximum number of arguments exceed
func runNginxTestHelper(t *testing.T, ctx context.Context, testName string,
	getUpstreamsFunc func(serviceInterface *nginxfakes.FakeNginxServiceInterface), instance *mpi.Instance,
	message *bus.Message, topic []string, err error,
) {
	t.Helper()

	t.Run(testName, func(tt *testing.T) {
		fakeNginxService := &nginxfakes.FakeNginxServiceInterface{}
		getUpstreamsFunc(fakeNginxService)

		if instance.GetInstanceMeta().GetInstanceId() != "e1374cb1-462d-3b6c-9f3b-f28332b5f10f" {
			fakeNginxService.InstanceReturns(instance)
		}

		messagePipe := busfakes.NewFakeMessagePipe()
		fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
		nginxPlugin := NewNginx(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
		nginxPlugin.nginxService = fakeNginxService

		registerErr := messagePipe.Register(2, []bus.Plugin{nginxPlugin})
		require.NoError(t, registerErr)

		nginxPlugin.messagePipe = messagePipe
		nginxPlugin.Process(ctx, message)

		assert.Equal(tt, topic[0], messagePipe.Messages()[0].Topic)

		response, ok := messagePipe.Messages()[0].Data.(*mpi.DataPlaneResponse)
		assert.True(tt, ok)

		if err != nil {
			assert.Equal(tt, err.Error(), response.GetCommandResponse().GetError())
			assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_FAILURE, response.GetCommandResponse().GetStatus())
		} else {
			assert.Empty(tt, response.GetCommandResponse().GetError())
			assert.Equal(tt, mpi.CommandResponse_COMMAND_STATUS_OK, response.GetCommandResponse().GetStatus())
		}
	})
}
