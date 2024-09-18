// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

var nginxConfigContext = &NginxConfigContext{
	StubStatus: "",
	PlusAPI:    "",
	InstanceID: "12333",
	Files: []*mpi.File{
		{
			FileMeta: protos.FileMeta("test1", "fef4e"),
		},
		{
			FileMeta: protos.FileMeta("test2", "vre8e"),
		},
	},
	AccessLogs: []*AccessLog{
		{
			Name:   "access1",
			Format: "something",
		},
	},
	ErrorLogs: []*ErrorLog{
		{
			Name: "error",
		},
	},
}

func TestNginxConfigContext_Equal(t *testing.T) {
	nginxConfigContextWithSameValues := *nginxConfigContext

	nginxConfigContextWithDifferentStubStatus := *nginxConfigContext
	nginxConfigContextWithDifferentStubStatus.StubStatus = "http://localhost:8080/stub_status"

	nginxConfigContextWithDifferentPlusAPI := *nginxConfigContext
	nginxConfigContextWithDifferentPlusAPI.PlusAPI = "http://localhost:8080/api"

	nginxConfigContextWithDifferentInstanceID := *nginxConfigContext
	nginxConfigContextWithDifferentInstanceID.InstanceID = "567"

	nginxConfigContextWithDifferentNumberOfFiles := *nginxConfigContext
	nginxConfigContextWithDifferentNumberOfFiles.Files = []*mpi.File{}

	nginxConfigContextWithDifferentFileHashes := *nginxConfigContext
	nginxConfigContextWithDifferentFileHashes.Files = []*mpi.File{
		{
			FileMeta: protos.FileMeta("test1", "fgregreger"),
		},
		{
			FileMeta: protos.FileMeta("test2", "vre8e"),
		},
	}

	nginxConfigContextWithDifferentAccessLogs := *nginxConfigContext
	nginxConfigContextWithDifferentAccessLogs.AccessLogs = []*AccessLog{}

	nginxConfigContextWithDifferentErrorLogs := *nginxConfigContext
	nginxConfigContextWithDifferentErrorLogs.ErrorLogs = []*ErrorLog{}

	assert.True(t, nginxConfigContext.Equal(&nginxConfigContextWithSameValues))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentStubStatus))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentPlusAPI))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentInstanceID))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentNumberOfFiles))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentFileHashes))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentAccessLogs))
	assert.False(t, nginxConfigContext.Equal(&nginxConfigContextWithDifferentErrorLogs))
}
