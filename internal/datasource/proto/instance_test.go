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
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			UpdateNginxInstanceRuntime(test.instance, test.nginxConfigContext)
			if test.name == "Test 2: Plus Instance" {
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
			} else {
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
