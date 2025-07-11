// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/pkg/id"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/require"
)

const timeout = 5 * time.Second

func TestNginxAppProtectInstanceWatcher_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedInstance := protos.NginxAppProtectInstance()

	versionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer os.Remove(versionFile.Name())
	defer versionFile.Close()

	_, err := versionFile.WriteString("5.144.0")
	require.NoError(t, err)

	// Instance ID is generated based on version file path
	expectedInstance.GetInstanceMeta().InstanceId = id.Generate(versionFile.Name())

	releaseFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "release")
	defer helpers.RemoveFileWithErrorCheck(t, releaseFile.Name())
	defer releaseFile.Close()

	_, err = releaseFile.WriteString("4.11.0")
	require.NoError(t, err)

	attackSignatureVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, attackSignatureVersionFile.Name())
	defer attackSignatureVersionFile.Close()

	_, err = attackSignatureVersionFile.WriteString("2024.11.28")
	require.NoError(t, err)

	threatCampaignVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "version")
	defer helpers.RemoveFileWithErrorCheck(t, threatCampaignVersionFile.Name())
	defer threatCampaignVersionFile.Close()

	_, err = threatCampaignVersionFile.WriteString("2024.12.02")
	require.NoError(t, err)

	enforcerEngineVersionFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "enforcer_version")
	defer helpers.RemoveFileWithErrorCheck(t, enforcerEngineVersionFile.Name())
	defer enforcerEngineVersionFile.Close()

	_, err = enforcerEngineVersionFile.WriteString("5.113.0")
	require.NoError(t, err)

	versionFilePath = versionFile.Name()
	releaseFilePath = releaseFile.Name()
	attackSignatureVersionFilePath = attackSignatureVersionFile.Name()
	threatCampaignVersionFilePath = threatCampaignVersionFile.Name()
	enforcerEngineVersionFilePath = enforcerEngineVersionFile.Name()

	versionFiles = []string{
		versionFilePath,
		releaseFilePath,
		attackSignatureVersionFilePath,
		threatCampaignVersionFilePath,
		enforcerEngineVersionFilePath,
	}

	instancesChannel := make(chan InstanceUpdatesMessage)

	nginxAppProtectInstanceWatcher := NewNginxAppProtectInstanceWatcher(
		&config.Config{
			Watchers: &config.Watchers{
				InstanceWatcher: config.InstanceWatcher{
					MonitoringFrequency: 200 * time.Millisecond,
				},
			},
		},
	)

	go nginxAppProtectInstanceWatcher.Watch(ctx, instancesChannel)

	t.Run("Test 1: New instance", func(t *testing.T) {
		select {
		case instanceUpdates := <-instancesChannel:
			assert.Len(t, instanceUpdates.InstanceUpdates.NewInstances, 1)
			assert.Empty(t, instanceUpdates.InstanceUpdates.UpdatedInstances)
			assert.Empty(t, instanceUpdates.InstanceUpdates.DeletedInstances)
			assert.Truef(
				t,
				proto.Equal(instanceUpdates.InstanceUpdates.NewInstances[0], expectedInstance),
				"expected %s, actual %s", expectedInstance, instanceUpdates.InstanceUpdates.NewInstances[0],
			)
		case <-time.After(timeout):
			t.Fatalf("Timed out waiting for instance updates")
		}
	})
	t.Run("Test 2: Update instance", func(t *testing.T) {
		_, err = enforcerEngineVersionFile.WriteAt([]byte("6.113.0"), 0)
		require.NoError(t, err)

		expectedInstance.GetInstanceRuntime().GetNginxAppProtectRuntimeInfo().EnforcerEngineVersion = "6.113.0"

		select {
		case instanceUpdates := <-instancesChannel:
			assert.Len(t, instanceUpdates.InstanceUpdates.UpdatedInstances, 1)
			assert.Empty(t, instanceUpdates.InstanceUpdates.NewInstances)
			assert.Empty(t, instanceUpdates.InstanceUpdates.DeletedInstances)
			assert.Truef(
				t,
				proto.Equal(instanceUpdates.InstanceUpdates.UpdatedInstances[0], expectedInstance),
				"expected %s, actual %s", expectedInstance, instanceUpdates.InstanceUpdates.UpdatedInstances[0],
			)
		case <-time.After(timeout):
			t.Fatalf("Timed out waiting for instance updates")
		}
	})
	t.Run("Test 3: Delete instance", func(t *testing.T) {
		helpers.RemoveFileWithErrorCheck(t, versionFile.Name())
		closeErr := versionFile.Close()
		require.NoError(t, closeErr)

		select {
		case instanceUpdates := <-instancesChannel:
			assert.Len(t, instanceUpdates.InstanceUpdates.DeletedInstances, 1)
			assert.Empty(t, instanceUpdates.InstanceUpdates.NewInstances)
			assert.Empty(t, instanceUpdates.InstanceUpdates.UpdatedInstances)
			assert.Truef(
				t,
				proto.Equal(instanceUpdates.InstanceUpdates.DeletedInstances[0], expectedInstance),
				"expected %s, actual %s", expectedInstance, instanceUpdates.InstanceUpdates.DeletedInstances[0],
			)
		case <-time.After(timeout):
			t.Fatalf("Timed out waiting for instance updates")
		}
	})
}
