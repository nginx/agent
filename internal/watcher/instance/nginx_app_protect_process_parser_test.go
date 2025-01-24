// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"os"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNginxAppProtectProcessParser_Parse(t *testing.T) {
	ctx := context.Background()

	expectedInstance := &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   "ca22d03b-06a4-3a2c-aa81-a6c4dd042ff4",
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX_APP_PROTECT,
			Version:      "5.144.0",
		},
		InstanceConfig: &mpi.InstanceConfig{},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  1111,
			BinaryPath: "/usr/share/ts/bin/bd-socket-plugin",
			Details: &mpi.InstanceRuntime_NginxAppProtectRuntimeInfo{
				NginxAppProtectRuntimeInfo: &mpi.NGINXAppProtectRuntimeInfo{
					Release:                "4.11.0",
					AttackSignatureVersion: "2024.11.28",
					ThreatCampaignVersion:  "2024.12.02",
				},
			},
			InstanceChildren: make([]*mpi.InstanceChild, 0),
		},
	}

	processes := []*model.Process{
		{
			PID:  789,
			PPID: 1234,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		{
			PID:  567,
			PPID: 1234,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		{
			PID:  1234,
			PPID: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  exePath,
		},
		{
			PID:  1111,
			PPID: 1,
			Name: "bd-socket-plugin",
			Cmd:  "/usr/share/ts/bin/bd-socket-plugin tmm_count 4 no_static_config",
			Exe:  "/usr/share/ts/bin/bd-socket-plugin",
		},
	}

	versionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, versionFile.Name())

	_, err := versionFile.WriteString("5.144.0")
	require.NoError(t, err)

	releaseFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "release")
	defer helpers.RemoveFileWithErrorCheck(t, releaseFile.Name())

	_, err = releaseFile.WriteString("4.11.0")
	require.NoError(t, err)

	attackSignatureVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, attackSignatureVersionFile.Name())

	_, err = attackSignatureVersionFile.WriteString("2024.11.28")
	require.NoError(t, err)

	threatCampaignVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, threatCampaignVersionFile.Name())

	_, err = threatCampaignVersionFile.WriteString("2024.12.02")
	require.NoError(t, err)

	nginxAppProtectProcessParser := NewNginxAppProtectProcessParser()
	nginxAppProtectProcessParser.versionFilePath = versionFile.Name()
	nginxAppProtectProcessParser.releaseFilePath = releaseFile.Name()
	nginxAppProtectProcessParser.attackSignatureVersionFilePath = attackSignatureVersionFile.Name()
	nginxAppProtectProcessParser.threatCampaignVersionFilePath = threatCampaignVersionFile.Name()

	instances := nginxAppProtectProcessParser.Parse(ctx, processes)

	assert.Len(t, instances, 1)

	assert.Truef(
		t,
		proto.Equal(instances["ca22d03b-06a4-3a2c-aa81-a6c4dd042ff4"], expectedInstance),
		"expected %s, actual %s", expectedInstance, instances["ca22d03b-06a4-3a2c-aa81-a6c4dd042ff4"],
	)
}
