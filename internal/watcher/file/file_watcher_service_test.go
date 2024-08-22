// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	directoryPermissions = 0o700
)

func TestFileWatcherService_NewFileWatcherService(t *testing.T) {
	fileWatcherService := NewFileWatcherService(types.AgentConfig())

	assert.Empty(t, fileWatcherService.directoriesBeingWatched)
	assert.True(t, fileWatcherService.enabled.Load())
	assert.False(t, fileWatcherService.filesChanged.Load())
}

func TestFileWatcherService_SetEnabled(t *testing.T) {
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	assert.True(t, fileWatcherService.enabled.Load())

	fileWatcherService.SetEnabled(false)
	assert.False(t, fileWatcherService.enabled.Load())

	fileWatcherService.SetEnabled(true)
	assert.True(t, fileWatcherService.enabled.Load())
}

func TestFileWatcherService_addWatcher(t *testing.T) {
	ctx := context.Background()
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fileWatcherService.watcher = watcher

	tempDir := os.TempDir()
	testDirectory := tempDir + "test_dir"
	err = os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.Remove(testDirectory)

	info, err := os.Stat(testDirectory)
	require.NoError(t, err)

	fileWatcherService.addWatcher(ctx, testDirectory, info)

	value, ok := fileWatcherService.directoriesBeingWatched.Load(testDirectory)
	assert.True(t, ok)
	boolValue, ok := value.(bool)
	assert.True(t, ok)
	assert.True(t, boolValue)
}

func TestFileWatcherService_addWatcher_Error(t *testing.T) {
	ctx := context.Background()
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fileWatcherService.watcher = watcher

	tempDir := os.TempDir()
	testDirectory := tempDir + "test_dir"
	err = os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	info, err := os.Stat(testDirectory)
	require.NoError(t, err)

	// Delete directory to cause the addWatcher function to fail
	err = os.Remove(testDirectory)
	require.NoError(t, err)

	fileWatcherService.addWatcher(ctx, testDirectory, info)

	value, ok := fileWatcherService.directoriesBeingWatched.Load(testDirectory)
	assert.True(t, ok)
	boolValue, ok := value.(bool)
	assert.True(t, ok)
	assert.False(t, boolValue)
	assert.True(t, ok)
}

func TestFileWatcherService_removeWatcher(t *testing.T) {
	ctx := context.Background()
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fileWatcherService.watcher = watcher

	tempDir := os.TempDir()
	testDirectory := tempDir + "test_dir"
	err = os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.Remove(testDirectory)

	err = fileWatcherService.watcher.Add(testDirectory)
	require.NoError(t, err)
	fileWatcherService.directoriesBeingWatched.Store(testDirectory, true)

	fileWatcherService.removeWatcher(ctx, testDirectory)

	value, ok := fileWatcherService.directoriesBeingWatched.Load(testDirectory)
	assert.Nil(t, value)
	assert.False(t, ok)
}

func TestFileWatcherService_isEventSkippable(t *testing.T) {
	assert.False(t, isEventSkippable(fsnotify.Event{Name: "test.conf"}))
	assert.True(t, isEventSkippable(fsnotify.Event{Name: "test.swp"}))
	assert.True(t, isEventSkippable(fsnotify.Event{Name: "test.swx"}))
	assert.True(t, isEventSkippable(fsnotify.Event{Name: "test.conf~"}))
}

func TestFileWatcherService_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := os.TempDir()
	testDirectory := tempDir + "test_dir"
	os.RemoveAll(testDirectory)
	err := os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.RemoveAll(testDirectory)

	agentConfig := types.AgentConfig()
	agentConfig.Watchers.FileWatcher.MonitoringFrequency = 100 * time.Millisecond
	agentConfig.AllowedDirectories = []string{testDirectory}

	channel := make(chan FileUpdateMessage)

	fileWatcherService := NewFileWatcherService(agentConfig)
	go fileWatcherService.Watch(ctx, channel)

	time.Sleep(100 * time.Millisecond)

	file, err := os.CreateTemp(testDirectory, "test.conf")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fileUpdate := <-channel
	assert.NotNil(t, fileUpdate.CorrelationID)

	skippableFile, err := os.CreateTemp(testDirectory, "*test.conf.swp")
	require.NoError(t, err)
	defer os.Remove(skippableFile.Name())

	select {
	case <-channel:
		t.Fatalf("Expected file to be skipped")
	case <-time.After(150 * time.Millisecond):
		return
	}
}
