// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/pkg/host/hostfakes"

	"github.com/nginx/agent/v3/internal/datasource/config/configfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginxinc/nginx-plus-go-client/v2/client"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nginx/agent/v3/internal/resource/resourcefakes"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceService_AddInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		resource     *v1.Resource
		instanceList []*v1.Instance
	}{
		{
			name: "Test 1: Add One Instance",
			instanceList: []*v1.Instance{
				protos.NginxOssInstance([]string{}),
			},
			resource: protos.HostResource(),
		},
		{
			name: "Test 2: Add Multiple Instance",
			instanceList: []*v1.Instance{
				protos.NginxOssInstance([]string{}),
				protos.NginxPlusInstance([]string{}),
			},
			resource: &v1.Resource{
				ResourceId: protos.HostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.NginxOssInstance([]string{}),
					protos.NginxPlusInstance([]string{}),
				},
				Info: protos.HostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resource := resourceService.AddInstances(test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_UpdateInstance(t *testing.T) {
	ctx := context.Background()

	updatedInstance := &v1.Instance{
		InstanceConfig: protos.NginxOssInstance([]string{}).GetInstanceConfig(),
		InstanceMeta:   protos.NginxOssInstance([]string{}).GetInstanceMeta(),
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  56789,
			BinaryPath: protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetBinaryPath(),
			ConfigPath: protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetConfigPath(),
			Details:    protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetDetails(),
		},
	}

	tests := []struct {
		name         string
		resource     *v1.Resource
		instanceList []*v1.Instance
	}{
		{
			name: "Test 1: Update Instances",
			instanceList: []*v1.Instance{
				updatedInstance,
			},
			resource: &v1.Resource{
				ResourceId: protos.HostResource().GetResourceId(),
				Instances: []*v1.Instance{
					updatedInstance,
				},
				Info: protos.HostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = []*v1.Instance{protos.NginxOssInstance([]string{})}
			resource := resourceService.UpdateInstances(ctx, test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_DeleteInstance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		err          error
		resource     *v1.Resource
		instanceList []*v1.Instance
	}{
		{
			name: "Test 1: Update Instances",
			instanceList: []*v1.Instance{
				protos.NginxPlusInstance([]string{}),
			},
			resource: &v1.Resource{
				ResourceId: protos.HostResource().GetResourceId(),
				Instances: []*v1.Instance{
					protos.NginxOssInstance([]string{}),
				},
				Info: protos.HostResource().GetInfo(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = []*v1.Instance{
				protos.NginxOssInstance([]string{}),
				protos.NginxPlusInstance([]string{}),
			}
			resource := resourceService.DeleteInstances(ctx, test.instanceList)
			assert.Equal(tt, test.resource.GetInstances(), resource.GetInstances())
		})
	}
}

func TestResourceService_Instance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		result    *v1.Instance
		name      string
		instances []*v1.Instance
	}{
		{
			name: "Test 1: instance found",
			instances: []*v1.Instance{
				protos.NginxOssInstance([]string{}),
				protos.NginxPlusInstance([]string{}),
			},
			result: protos.NginxPlusInstance([]string{}),
		},
		{
			name: "Test 2: instance not found",
			instances: []*v1.Instance{
				protos.NginxOssInstance([]string{}),
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = test.instances
			instance := resourceService.Instance(protos.NginxPlusInstance([]string{}).
				GetInstanceMeta().GetInstanceId())
			assert.Equal(tt, test.result, instance)
		})
	}
}

func TestResourceService_GetResource(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedResource *v1.Resource
		isContainer      bool
	}{
		{
			isContainer:      true,
			expectedResource: protos.ContainerizedResource(),
		},
		{
			isContainer:      false,
			expectedResource: protos.HostResource(),
		},
	}
	for _, tc := range testCases {
		mockInfo := &hostfakes.FakeInfoInterface{}
		if tc.isContainer {
			mockInfo.ContainerInfoReturns(
				&v1.Resource_ContainerInfo{
					ContainerInfo: tc.expectedResource.GetContainerInfo(),
				}, nil,
			)
		} else {
			mockInfo.HostInfoReturns(
				&v1.Resource_HostInfo{
					HostInfo: tc.expectedResource.GetHostInfo(),
				}, nil,
			)
		}

		mockInfo.IsContainerReturns(tc.isContainer, nil)

		resourceService := NewResourceService(ctx, types.AgentConfig())
		resourceService.info = mockInfo
		resourceService.resource = tc.expectedResource

		resourceService.updateResourceInfo(ctx)
		assert.Equal(t, tc.expectedResource.GetResourceId(), resourceService.resource.GetResourceId())
		assert.Empty(t, resourceService.resource.GetInstances())

		if tc.isContainer {
			assert.Equal(t, tc.expectedResource.GetContainerInfo(), resourceService.resource.GetContainerInfo())
		} else {
			assert.Equal(t, tc.expectedResource.GetHostInfo(), resourceService.resource.GetHostInfo())
		}
	}
}

func TestResourceService_createPlusClient(t *testing.T) {
	// Create a temporary file for testing CA certificate
	tempDir := t.TempDir()
	caFile := filepath.Join(tempDir, "test-ca.crt")

	err := os.WriteFile(caFile, []byte("-----BEGIN CERTIFICATE-----\nMII...\n-----END CERTIFICATE-----"), 0o600)
	require.NoError(t, err)

	createPlusInstanceWithApis := func(details []*v1.APIDetails) *v1.Instance {
		inst := protos.NginxPlusInstance([]string{})
		if inst.GetInstanceRuntime().GetNginxPlusRuntimeInfo() == nil {
			inst.InstanceRuntime.Details = &v1.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &v1.NGINXPlusRuntimeInfo{},
			}
		}
		inst.InstanceRuntime.GetNginxPlusRuntimeInfo().PlusApis = details

		return inst
	}

	instanceWithAPI := createPlusInstanceWithApis([]*v1.APIDetails{
		{Location: "/api", Listen: "localhost:80"},
	})

	instanceWithUnixAPI := createPlusInstanceWithApis([]*v1.APIDetails{
		{Listen: "unix:/var/run/nginx-status.sock", Location: "/api"},
	})

	instanceWithCACert := createPlusInstanceWithApis([]*v1.APIDetails{
		{Location: "/api", Listen: "localhost:443", Ca: caFile},
	})

	var logBuffer bytes.Buffer

	tempHandler := slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	tempLogger := slog.New(tempHandler)

	ctx := context.Background()
	tests := []struct {
		err         error
		instance    *v1.Instance
		name        string
		expectedLog string
	}{
		{
			name:     "Test 1: Create Plus Client",
			instance: instanceWithAPI,
			err:      nil,
		},
		{
			name:     "Test 2: Create Plus Client, Unix",
			instance: instanceWithUnixAPI,
			err:      nil,
		},
		{
			name:     "Test 3: Create Plus Client with CA Certificate",
			instance: instanceWithCACert,
			err:      nil,
		},
		{
			name:     "Test 4: Fail Creating Client - API not Configured",
			instance: protos.NginxPlusInstance([]string{}),
			err:      errors.New("failed to preform API action, NGINX Plus API is not configured"),
		},
		{
			name: "Test 5: Fallback to First API (No Write-Enabled)",
			instance: createPlusInstanceWithApis([]*v1.APIDetails{
				{Location: "/read1", Listen: "localhost:80"},
				{Location: "/read2", Listen: "localhost:8080"},
			}),
			err:         nil,
			expectedLog: "No write-enabled NGINX Plus API found. Write operations may fail.",
		},
		{
			name: "Test 6: Prioritize Write-Enabled API",
			instance: createPlusInstanceWithApis([]*v1.APIDetails{
				{Location: "/read", Listen: "localhost:80"},
				{Location: "/write", Listen: "localhost:8080", WriteEnabled: true},
			}),
			err:         nil,
			expectedLog: "Selected write-enabled NGINX Plus API for action",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			logBuffer.Reset()
			originalLogger := slog.Default()
			slog.SetDefault(tempLogger)
			defer slog.SetDefault(originalLogger)
			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceService.resource.Instances = []*v1.Instance{
				protos.NginxOssInstance([]string{}),
				protos.NginxPlusInstance([]string{}),
			}

			_, clientErr := resourceService.createPlusClient(ctx, test.instance)
			if test.err != nil {
				require.Error(tt, clientErr)
				assert.Contains(tt, clientErr.Error(), test.err.Error())
			} else {
				require.NoError(tt, clientErr)
				// For the CA cert test, we can't easily verify the internal http.Client configuration
				// without exporting it or adding test hooks, so we'll just verify no error is returned
			}
			if test.expectedLog != "" {
				logOutput := logBuffer.String()
				assert.Contains(tt, logOutput, test.expectedLog)
			}
		})
	}
}

func TestResourceService_ApplyConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		instanceID  string
		reloadErr   error
		validateErr error
		expected    error
		name        string
	}{
		{
			name:        "Test 1: Successful reload",
			instanceID:  protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			reloadErr:   nil,
			validateErr: nil,
			expected:    nil,
		},
		{
			name:        "Test 2: Failed reload",
			instanceID:  protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			reloadErr:   errors.New("something went wrong"),
			validateErr: nil,
			expected:    fmt.Errorf("failed to reload NGINX %w", errors.New("something went wrong")),
		},
		{
			name:        "Test 3: Failed validate",
			instanceID:  protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			reloadErr:   nil,
			validateErr: errors.New("something went wrong"),
			expected:    fmt.Errorf("failed validating config %w", errors.New("something went wrong")),
		},
		{
			name:        "Test 4: Unknown instance ID",
			instanceID:  "unknown",
			reloadErr:   nil,
			validateErr: nil,
			expected:    errors.New("instance unknown not found"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			instanceOp := &resourcefakes.FakeInstanceOperator{}

			instanceOp.ReloadReturns(test.reloadErr)
			instanceOp.ValidateReturns(test.validateErr)

			nginxParser := configfakes.FakeConfigParser{}

			nginxParser.ParseReturns(&model.NginxConfigContext{
				StubStatus:       &model.APIDetails{},
				PlusAPI:          &model.APIDetails{},
				InstanceID:       test.instanceID,
				Files:            nil,
				AccessLogs:       nil,
				ErrorLogs:        nil,
				NAPSysLogServers: []string{},
			}, nil)

			nginxParser.FindStubStatusAPIReturns(&model.APIDetails{
				URL:      "",
				Listen:   "",
				Location: "",
				Ca:       "",
			})

			nginxParser.FindPlusAPIReturns(&model.APIDetails{
				URL:      "",
				Listen:   "",
				Location: "",
				Ca:       "",
			})

			resourceService := NewResourceService(ctx, types.AgentConfig())
			resourceOpMap := make(map[string]instanceOperator)
			resourceOpMap[protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()] = instanceOp
			resourceService.instanceOperators = resourceOpMap
			resourceService.nginxConfigParser = &nginxParser

			instance := protos.NginxOssInstance([]string{})
			instances := []*v1.Instance{
				instance,
			}
			resourceService.resource.Instances = instances

			_, reloadError := resourceService.ApplyConfig(ctx, test.instanceID)
			assert.Equal(t, test.expected, reloadError)
		})
	}
}

func Test_convertToUpstreamServer(t *testing.T) {
	expectedMax := 2
	expectedFails := 0
	expectedBackup := true
	expected := []client.UpstreamServer{
		{
			MaxConns: &expectedMax,
			MaxFails: &expectedFails,
			Backup:   &expectedBackup,
			Server:   "test_server",
		},
		{
			MaxConns: &expectedMax,
			MaxFails: &expectedFails,
			Backup:   &expectedBackup,
			Server:   "test_server2",
		},
	}

	runUpstreamServerConversionTest(t, expected, func(data []*structpb.Struct) interface{} {
		return convertToUpstreamServer(data)
	})
}

func Test_convertToStreamUpstreamServer(t *testing.T) {
	expectedMax := 2
	expectedFails := 0
	expectedBackup := true
	expected := []client.StreamUpstreamServer{
		{
			MaxConns: &expectedMax,
			MaxFails: &expectedFails,
			Backup:   &expectedBackup,
			Server:   "test_server",
		},
		{
			MaxConns: &expectedMax,
			MaxFails: &expectedFails,
			Backup:   &expectedBackup,
			Server:   "test_server2",
		},
	}

	runUpstreamServerConversionTest(t, expected, func(data []*structpb.Struct) interface{} {
		return convertToStreamUpstreamServer(data)
	})
}

//nolint:lll // this need to be in one line else we will get gofumpt error
func runUpstreamServerConversionTest(t *testing.T, expectedUpstreamServers interface{}, conversionFunc func([]*structpb.Struct) interface{}) {
	t.Helper()
	expectedMax := 2
	expectedFails := 0
	expectedBackup := true

	testData := []*structpb.Struct{
		{
			Fields: map[string]*structpb.Value{
				"max_conns": structpb.NewNumberValue(float64(expectedMax)),
				"max_fails": structpb.NewNumberValue(float64(expectedFails)),
				"backup":    structpb.NewBoolValue(expectedBackup),
				"server":    structpb.NewStringValue("test_server"),
			},
		},
		{
			Fields: map[string]*structpb.Value{
				"max_conns": structpb.NewNumberValue(float64(expectedMax)),
				"max_fails": structpb.NewNumberValue(float64(expectedFails)),
				"backup":    structpb.NewBoolValue(expectedBackup),
				"server":    structpb.NewStringValue("test_server2"),
			},
		},
	}

	result := conversionFunc(testData)
	assert.Equal(t, expectedUpstreamServers, result)
}
