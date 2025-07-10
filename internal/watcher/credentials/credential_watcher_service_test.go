// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package credentials

import (
	"context"
	"errors"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialWatcherService_TestNewCredentialWatcherService(t *testing.T) {
	credentialWatcherService := NewCredentialWatcherService(types.AgentConfig(), model.Command)

	assert.Empty(t, credentialWatcherService.filesBeingWatched)
	assert.False(t, credentialWatcherService.filesChanged.Load())
}

func TestCredentialWatcherService_Watch(t *testing.T) {
	ctx := context.Background()
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	cws.watcher = watcher

	cuc := make(chan CredentialUpdateMessage)

	name := path.Join(os.TempDir(), "test_file")
	_, err = os.Create(name)
	require.NoError(t, err)
	defer os.Remove(name)

	cws.agentConfig.Command.Auth.TokenPath = name
	cws.filesChanged.Store(true)
	go cws.Watch(ctx, cuc)

	select {
	case <-ctx.Done():
		t.Error("context done")
	case <-cuc:
		assert.True(t, cws.isWatching(name))
	case <-time.After(2 * monitoringInterval):
		t.Error("Timed out waiting for credential watch")
	}

	func() {
		cws.watcher.Errors <- errors.New("watch error")
	}()
}

func TestCredentialWatcherService_isWatching(t *testing.T) {
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	assert.False(t, cws.isWatching("test-file"))
	cws.filesBeingWatched.Store("test-file", true)
	assert.True(t, cws.isWatching("test-file"))
	cws.filesBeingWatched.Store("test-file", false)
	assert.False(t, cws.isWatching("test-file"))
}

func TestCredentialWatcherService_isEventSkippable(t *testing.T) {
	assert.False(t, isEventSkippable(fsnotify.Event{Name: "testWriteEvent", Op: fsnotify.Write}))
	assert.True(t, isEventSkippable(fsnotify.Event{Name: "", Op: 0}))
	assert.True(t, isEventSkippable(fsnotify.Event{Name: "", Op: fsnotify.Write}))
	assert.True(t, isEventSkippable(fsnotify.Event{Op: fsnotify.Chmod}))
	assert.True(t, isEventSkippable(fsnotify.Event{Op: fsnotify.Rename}))
	assert.True(t, isEventSkippable(fsnotify.Event{Op: fsnotify.Create}))
}

func TestCredentialWatcherService_addWatcher(t *testing.T) {
	ctx := context.Background()
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	cws.watcher = watcher

	name := path.Join(os.TempDir(), "test_file")
	_, err = os.Create(name)
	require.NoError(t, err)
	defer os.Remove(name)

	cws.addWatcher(ctx, name)
	require.True(t, cws.isWatching(name))

	cws.addWatcher(ctx, name)
	require.True(t, cws.isWatching(name))

	name = path.Join(os.TempDir(), "noexist_file")
	cws.addWatcher(ctx, name)
	require.False(t, cws.isWatching(name))
}

func TestCredentialWatcherService_watchFiles(t *testing.T) {
	var files []string

	ctx := context.Background()
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	cws.watcher = watcher

	files = append(files, path.Join(os.TempDir(), "test_file1"))
	files = append(files, path.Join(os.TempDir(), "test_file2"))
	files = append(files, path.Join(os.TempDir(), "test_file3"))

	for _, file := range files {
		_, err = os.Create(file)
		require.NoError(t, err)
	}

	cws.watchFiles(ctx, files)
	require.True(t, cws.isWatching(path.Join(os.TempDir(), "test_file1")))
	require.True(t, cws.isWatching(path.Join(os.TempDir(), "test_file2")))
	require.True(t, cws.isWatching(path.Join(os.TempDir(), "test_file3")))

	for _, file := range files {
		err = os.Remove(file)
		cws.filesBeingWatched.Delete(file)
		require.NoError(t, err)
	}

	require.False(t, cws.isWatching(path.Join(os.TempDir(), "test_file1")))
	require.False(t, cws.isWatching(path.Join(os.TempDir(), "test_file2")))
	require.False(t, cws.isWatching(path.Join(os.TempDir(), "test_file3")))
}

func TestCredentialWatcherService_checkForUpdates(t *testing.T) {
	ctx := context.Background()
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	cws.watcher = watcher

	name := path.Join(os.TempDir(), "test_file")
	_, err = os.Create(name)
	require.NoError(t, err)
	cws.addWatcher(ctx, name)
	require.True(t, cws.isWatching(name))

	cws.filesChanged.Store(true)
	ch := make(chan CredentialUpdateMessage)
	go cws.checkForUpdates(ctx, ch)

	select {
	case <-ctx.Done():
		t.Error(ctx.Err())
	case cu := <-ch:
		t.Logf("check for update success %v", cu)
	case <-time.After(2 * monitoringInterval):
		t.Error("timeout waiting for update")
	}
}

func TestCredentialWatcherService_handleEvent(t *testing.T) {
	ctx := context.Background()
	cws := NewCredentialWatcherService(types.AgentConfig(), model.Command)
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	cws.watcher = watcher

	cws.handleEvent(ctx, fsnotify.Event{Name: "test-write", Op: fsnotify.Chmod})
	assert.False(t, cws.filesChanged.Load())
	cws.handleEvent(ctx, fsnotify.Event{Name: "test-create", Op: fsnotify.Create})
	assert.False(t, cws.filesChanged.Load())
	cws.handleEvent(ctx, fsnotify.Event{Name: "test-remove", Op: fsnotify.Remove})
	assert.True(t, cws.filesChanged.Load())
	cws.handleEvent(ctx, fsnotify.Event{Name: "test-rename", Op: fsnotify.Rename})
	assert.True(t, cws.filesChanged.Load())
	cws.handleEvent(ctx, fsnotify.Event{Name: "test-write", Op: fsnotify.Write})
	assert.True(t, cws.filesChanged.Load())
}

func Test_credentialPaths(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		want        []string
	}{
		{
			name:        "Test 1: Returns expected paths when Auth TokenPath is set",
			agentConfig: types.AgentConfig(),
			want: []string{
				"/tmp/token",
				"ca.pem",
				"cert.pem",
				"key.pem",
			},
		},
		{
			name: "Test 2: Returns empty slice when Auth TokenPath is not set",
			agentConfig: &config.Config{
				Command: &config.Command{
					Server: nil,
					Auth:   nil,
					TLS:    nil,
				},
			},
			want: nil,
		},
		{
			name: "Test 3: Add TLS paths if Command TLS is set",
			agentConfig: &config.Config{
				Command: &config.Command{
					Server: nil,
					Auth:   nil,
					TLS: &config.TLSConfig{
						Cert:       "/tmp-ca",
						Key:        "/tmp-token",
						Ca:         "/tmp-key",
						ServerName: "my-server",
						SkipVerify: false,
					},
				},
			},
			want: []string{
				"/tmp-key",
				"/tmp-ca",
				"/tmp-token",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, credentialPaths(tt.agentConfig.Command), "credentialPaths(%v)", tt.agentConfig)
		})
	}
}
