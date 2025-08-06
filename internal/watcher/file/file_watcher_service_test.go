// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/stub"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	directoryPermissions = 0o700
)

func TestFileWatcherService_NewFileWatcherService(t *testing.T) {
	fileWatcherService := NewFileWatcherService(types.AgentConfig())

	assert.Empty(t, fileWatcherService.directoriesToWatch)
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

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")
	err = os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.Remove(testDirectory)

	require.NoError(t, fileWatcherService.addWatcher(ctx, testDirectory))

	directoriesBeingWatched := fileWatcherService.watcher.WatchList()
	assert.Len(t, directoriesBeingWatched, 1)
	assert.Equal(t, testDirectory, directoriesBeingWatched[0])
}

func TestFileWatcherService_addWatcher_Error(t *testing.T) {
	ctx := context.Background()
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fileWatcherService.watcher = watcher

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")

	require.Error(t, fileWatcherService.addWatcher(ctx, testDirectory))

	directoriesBeingWatched := fileWatcherService.watcher.WatchList()
	assert.Empty(t, directoriesBeingWatched)
}

func TestFileWatcherService_removeWatcher(t *testing.T) {
	ctx := context.Background()
	fileWatcherService := NewFileWatcherService(types.AgentConfig())
	watcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	fileWatcherService.watcher = watcher

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")
	err = os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.Remove(testDirectory)

	err = fileWatcherService.watcher.Add(testDirectory)
	require.NoError(t, err)

	fileWatcherService.removeWatcher(ctx, testDirectory)

	logBuf := &bytes.Buffer{}
	defer logBuf.Reset()
	stub.StubLoggerWith(logBuf)

	fileWatcherService.removeWatcher(ctx, testDirectory)

	helpers.ValidateLog(t, "Failed to remove file watcher", logBuf)
}

func TestFileWatcherService_isEventSkippable(t *testing.T) {
	config := types.AgentConfig()
	config.Watchers.FileWatcher.ExcludeFiles = []string{"^/var/log/nginx/.*.log$", "\\.*swp$", "\\.*swx$", ".*~$"}
	fws := NewFileWatcherService(config)

	assert.False(t, fws.isEventSkippable(fsnotify.Event{Name: "test.conf"}))
	assert.True(t, fws.isEventSkippable(fsnotify.Event{Name: "test.swp"}))
	assert.True(t, fws.isEventSkippable(fsnotify.Event{Name: "test.swx"}))
	assert.True(t, fws.isEventSkippable(fsnotify.Event{Name: "test.conf~"}))
	assert.True(t, fws.isEventSkippable(fsnotify.Event{Name: "/var/log/nginx/access.log"}))
}

func TestFileWatcherService_isExcludedFile(t *testing.T) {
	excludeFiles := []string{"/var/log/nginx/access.log", "^.*(\\.log|.swx|~|.swp)$"}

	assert.True(t, isExcludedFile("/var/log/nginx/error.log", excludeFiles))
	assert.True(t, isExcludedFile("/var/log/nginx/error.swx", excludeFiles))
	assert.True(t, isExcludedFile("test.swp", excludeFiles))
	assert.True(t, isExcludedFile("/var/log/nginx/error~", excludeFiles))
	assert.True(t, isExcludedFile("/var/log/nginx/access.log", excludeFiles))
	assert.False(t, isExcludedFile("/etc/nginx/nginx.conf", excludeFiles))
	assert.False(t, isExcludedFile("/var/log/accesslog", excludeFiles))
}

func TestFileWatcherService_Update(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")
	err := os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.RemoveAll(testDirectory)

	agentConfig := types.AgentConfig()
	agentConfig.Watchers.FileWatcher.MonitoringFrequency = 100 * time.Millisecond
	agentConfig.AllowedDirectories = []string{testDirectory, "/unknown/directory"}

	fileWatcherService := NewFileWatcherService(agentConfig)

	t.Run("Test 1: watcher not initialized yet", func(t *testing.T) {
		fileWatcherService.Update(ctx, &model.NginxConfigContext{
			Includes: []string{filepath.Join(testDirectory, "*.conf")},
		})

		_, ok := fileWatcherService.directoriesToWatch[testDirectory]
		assert.True(t, ok)

		assert.Nil(t, fileWatcherService.watcher)
	})

	t.Run("Test 2: watcher initialized", func(t *testing.T) {
		watcher, newWatcherError := fsnotify.NewWatcher()
		require.NoError(t, newWatcherError)

		fileWatcherService.watcher = watcher

		fileWatcherService.Update(ctx, &model.NginxConfigContext{
			Includes: []string{filepath.Join(testDirectory, "*.conf")},
		})

		_, ok := fileWatcherService.directoriesToWatch[testDirectory]
		assert.True(t, ok)

		directoriesBeingWatched := fileWatcherService.watcher.WatchList()
		assert.Len(t, directoriesBeingWatched, 1)
		assert.Equal(t, testDirectory, directoriesBeingWatched[0])
	})

	t.Run("Test 3: remove watchers", func(t *testing.T) {
		fileWatcherService.Update(ctx, &model.NginxConfigContext{
			Includes: []string{},
		})

		assert.Empty(t, fileWatcherService.directoriesToWatch)

		directoriesBeingWatched := fileWatcherService.watcher.WatchList()
		assert.Empty(t, directoriesBeingWatched)
	})

	t.Run("Test 4: not allowed directory", func(t *testing.T) {
		fileWatcherService.Update(ctx, &model.NginxConfigContext{
			Files: []*mpi.File{
				{
					FileMeta: &mpi.FileMeta{
						Name: "/unknown/location/test.conf",
					},
				},
			},
		})

		_, ok := fileWatcherService.directoriesToWatch["/unknown/location/test.conf"]
		assert.False(t, ok)

		directoriesBeingWatched := fileWatcherService.watcher.WatchList()
		assert.Empty(t, directoriesBeingWatched)
	})
}

func TestFileWatcherService_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")
	err := os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.Remove(testDirectory)

	agentConfig := types.AgentConfig()
	agentConfig.Watchers.FileWatcher.MonitoringFrequency = 100 * time.Millisecond
	agentConfig.AllowedDirectories = []string{testDirectory, "/unknown/directory"}

	channel := make(chan FileUpdateMessage)

	fileWatcherService := NewFileWatcherService(agentConfig)
	go fileWatcherService.Watch(ctx, channel)

	time.Sleep(100 * time.Millisecond)

	fileWatcherService.Update(ctx, &model.NginxConfigContext{
		Includes: []string{filepath.Join(testDirectory, "*.conf")},
	})

	file, err := os.CreateTemp(testDirectory, "test.conf")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	t.Run("Test 1: File updated", func(t *testing.T) {
		// Check that directory is being watched
		assert.Eventually(t, func() bool {
			_, ok := fileWatcherService.directoriesToWatch[testDirectory]
			return ok
		}, 1*time.Second, 100*time.Millisecond)

		assert.Eventually(t, func() bool {
			directoriesBeingWatched := fileWatcherService.watcher.WatchList()
			return len(directoriesBeingWatched) == 1
		}, 1*time.Second, 100*time.Millisecond)

		select {
		case fileUpdate := <-channel:
			assert.NotNil(t, fileUpdate.CorrelationID)
		case <-time.After(150 * time.Millisecond):
			t.Fatalf("Expected file update event")
		}
	})

	t.Run("Test 2: Skippable file updated", func(t *testing.T) {
		skippableFile, skippableFileError := os.CreateTemp(testDirectory, "*test.conf.swp")
		require.NoError(t, skippableFileError)
		defer os.Remove(skippableFile.Name())

		select {
		case <-channel:
			t.Fatalf("Expected file to be skipped: %v", skippableFile.Name())
		case <-time.After(150 * time.Millisecond):
			return
		}
	})

	t.Run("Test 3: Directory deleted", func(t *testing.T) {
		dirDeleteError := os.RemoveAll(testDirectory)
		require.NoError(t, dirDeleteError)

		// Check that directory is no longer being watched
		assert.Eventually(t, func() bool {
			_, ok := fileWatcherService.directoriesToWatch[testDirectory]
			return ok
		}, 1*time.Second, 100*time.Millisecond)

		assert.Eventually(t, func() bool {
			directoriesBeingWatched := fileWatcherService.watcher.WatchList()
			return len(directoriesBeingWatched) == 0
		}, 1*time.Second, 100*time.Millisecond)
	})
}

func TestFileWatcherService_checkForUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempDir := t.TempDir()
	testDirectory := path.Join(tempDir, "test_dir")
	err := os.Mkdir(testDirectory, directoryPermissions)
	require.NoError(t, err)
	defer os.RemoveAll(testDirectory)

	agentConfig := types.AgentConfig()
	agentConfig.Watchers.FileWatcher.MonitoringFrequency = 100 * time.Millisecond
	agentConfig.AllowedDirectories = []string{testDirectory, "/unknown/directory"}

	channel := make(chan FileUpdateMessage)

	fileWatcherService := NewFileWatcherService(agentConfig)
	fileWatcherService.filesChanged.Store(true)
	assert.Nil(t, fileWatcherService.watcher)

	go fileWatcherService.checkForUpdates(ctx, channel)

	select {
	case fileUpdate := <-channel:
		assert.NotNil(t, fileUpdate.CorrelationID)
		assert.NotNil(t, fileWatcherService.watcher)
		assert.False(t, fileWatcherService.filesChanged.Load())
	case <-time.After(150 * time.Millisecond):
		t.Fatalf("Expected file update event")
	}
}
