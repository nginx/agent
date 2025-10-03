// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

var plusAPIs = []*model.APIDetails{
	{
		URL:          "http://127.0.0.1:8081/api",
		Listen:       "127.0.0.1:8081",
		Location:     "/api",
		WriteEnabled: false,
		Ca:           "",
	},
	{
		URL:          "unix:/var/run/nginx/api.sock",
		Listen:       "unix:/var/run/nginx/api.sock",
		Location:     "/api",
		WriteEnabled: true, // Crucial for selection logic
		Ca:           "/etc/certs/my_ca.pem",
	},
}

var nginxPlusConfigContextForUpdate = &model.NginxConfigContext{
	AccessLogs: []*model.AccessLog{
		{
			Name: "/usr/local/var/log/nginx/access.log",
		},
	},
	ErrorLogs: []*model.ErrorLog{
		{
			Name: "/usr/local/var/log/nginx/error.log",
		},
	},
	PlusAPI: plusAPIs[1],
	StubStatus: &model.APIDetails{
		URL:    "http://127.0.0.1:8081/status",
		Listen: "127.0.0.1:8081",
	},
	PlusAPIs: plusAPIs,
}

func convertAPIDetailsSliceForTest(modelAPIs []*model.APIDetails) []*mpi.APIDetails {
	if modelAPIs == nil {
		return nil
	}
	mpiAPIs := make([]*mpi.APIDetails, 0, len(modelAPIs))
	for _, api := range modelAPIs {
		mpiAPIs = append(mpiAPIs, convertToMpiAPIDetails(api))
	}

	return mpiAPIs
}

func TestInstanceWatcherService_updateNginxInstanceRuntime(t *testing.T) {
	nginxOSSConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		StubStatus: &model.APIDetails{
			URL:    "http://127.0.0.1:8081/api",
			Listen: "",
		},
	}

	nginxPlusConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		PlusAPI: &model.APIDetails{
			URL:    "http://127.0.0.1:8081/api",
			Listen: "",
		},
		StubStatus: &model.APIDetails{
			URL:    "http://127.0.0.1:8081/api",
			Listen: "",
		},
	}

	tests := []struct {
		nginxConfigContext *model.NginxConfigContext
		instance           *mpi.Instance
		name               string
	}{
		{
			name:               "Test 1: OSS Instance",
			nginxConfigContext: nginxOSSConfigContext,
			instance:           protos.NginxOssInstance([]string{}),
		},
		{
			name:               "Test 2: Plus Instance",
			nginxConfigContext: nginxPlusConfigContext,
			instance:           protos.NginxPlusInstance([]string{}),
		},
		{
			name:               "Test 3: Plus Instance - PlusAPIs Update",
			nginxConfigContext: nginxPlusConfigContextForUpdate,
			instance:           protos.NginxPlusInstance([]string{}),
		},
		{
			name:               "Test 4: Plus Instance - No Update Required",
			nginxConfigContext: nginxPlusConfigContextForUpdate,
			instance:           createPopulatedNginxPlusInstance(nginxPlusConfigContextForUpdate),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updatesRequired := UpdateNginxInstanceRuntime(test.instance, test.nginxConfigContext)
			switch test.name {
			case "Test 3: Plus Instance - PlusAPIs Update":
				assert.True(t, updatesRequired,
					"UpdateNginxInstanceRuntime should return true when PlusAPIs are updated")
				expectedAPIs := convertAPIDetailsSliceForTest(test.nginxConfigContext.PlusAPIs)
				assert.ElementsMatch(t, expectedAPIs,
					test.instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetPlusApis())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.WriteEnabled, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetWriteEnabled())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.Ca, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetCa())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.Listen, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetListen())
			case "Test 4: Plus Instance - No Update Required":
				assert.False(t, updatesRequired,
					"UpdateNginxInstanceRuntime should return false when runtime already matches config")

			case "Test 2: Plus Instance":
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus.Location, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetStubStatus().GetLocation())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.Location, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetLocation())
				assert.Equal(t, test.nginxConfigContext.StubStatus.Listen, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetStubStatus().GetListen())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.Listen, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetListen())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.WriteEnabled, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetWriteEnabled())
				assert.Equal(t, test.nginxConfigContext.PlusAPI.Ca, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi().GetCa())

			default:
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus.Location, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetStubStatus().GetLocation())
				assert.Equal(t, test.nginxConfigContext.StubStatus.Listen, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetStubStatus().GetListen())
			}
		})
	}
}

func createPopulatedNginxPlusInstance(configContext *model.NginxConfigContext) *mpi.Instance {
	instance := protos.NginxPlusInstance([]string{})
	runtimeInfo := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo()
	runtimeInfo.PlusApi.Listen = configContext.PlusAPI.Listen

	runtimeInfo.PlusApi.Location = configContext.PlusAPI.Location
	runtimeInfo.PlusApi.WriteEnabled = configContext.PlusAPI.WriteEnabled
	runtimeInfo.PlusApi.Ca = configContext.PlusAPI.Ca

	runtimeInfo.PlusApis = convertAPIDetailsSliceForTest(configContext.PlusAPIs)

	runtimeInfo.AccessLogs = model.ConvertAccessLogs(configContext.AccessLogs)
	runtimeInfo.ErrorLogs = model.ConvertErrorLogs(configContext.ErrorLogs)

	runtimeInfo.StubStatus.Listen = configContext.StubStatus.Listen
	runtimeInfo.StubStatus.Location = configContext.StubStatus.Location

	return instance
}
