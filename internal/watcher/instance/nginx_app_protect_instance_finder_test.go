// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"os"
	"testing"

	"github.com/nginx/agent/v3/pkg/id"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNginxAppProtectInstanceFinder_Find(t *testing.T) {
	ctx := context.Background()

	expectedInstance := protos.NginxAppProtectInstance()

	versionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, versionFile.Name())

	_, err := versionFile.WriteString("5.144.0")
	require.NoError(t, err)

	// Instance ID is generated based on version file path
	expectedInstance.GetInstanceMeta().InstanceId = id.Generate(versionFile.Name())

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

	enforcerEngineVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "enforcer_version")
	defer helpers.RemoveFileWithErrorCheck(t, enforcerEngineVersionFile.Name())

	_, err = enforcerEngineVersionFile.WriteString("5.113.0")
	require.NoError(t, err)

	nginxAppProtectInstanceFinder := NewNginxAppProtectInstanceFinder()
	nginxAppProtectInstanceFinder.versionFilePath = versionFile.Name()
	nginxAppProtectInstanceFinder.releaseFilePath = releaseFile.Name()
	nginxAppProtectInstanceFinder.attackSignatureVersionFilePath = attackSignatureVersionFile.Name()
	nginxAppProtectInstanceFinder.threatCampaignVersionFilePath = threatCampaignVersionFile.Name()
	nginxAppProtectInstanceFinder.enforcerEngineVersionFilePath = enforcerEngineVersionFile.Name()

	instance := nginxAppProtectInstanceFinder.Find(ctx)

	assert.Truef(
		t,
		proto.Equal(instance, expectedInstance),
		"expected %s, actual %s", expectedInstance, instance,
	)
}
