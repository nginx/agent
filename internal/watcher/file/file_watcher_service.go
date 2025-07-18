// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
)

var emptyEvent = fsnotify.Event{
	Name: "",
	Op:   0,
}

type FileUpdateMessage struct {
	CorrelationID slog.Attr
}

type FileWatcherService struct {
	agentConfig        *config.Config
	watcher            *fsnotify.Watcher
	filesChanged       *atomic.Bool
	enabled            *atomic.Bool
	directoriesToWatch map[string]struct{}
	mu                 sync.Mutex
}

func NewFileWatcherService(agentConfig *config.Config) *FileWatcherService {
	enabled := &atomic.Bool{}
	enabled.Store(true)

	filesChanged := &atomic.Bool{}
	filesChanged.Store(false)

	return &FileWatcherService{
		agentConfig:        agentConfig,
		directoriesToWatch: make(map[string]struct{}),
		enabled:            enabled,
		filesChanged:       filesChanged,
	}
}

func (fws *FileWatcherService) Watch(ctx context.Context, ch chan<- FileUpdateMessage) {
	monitoringFrequency := fws.agentConfig.Watchers.FileWatcher.MonitoringFrequency
	slog.DebugContext(ctx, "Starting file watcher monitoring", "monitoring_frequency", monitoringFrequency)

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create file watcher", "error", err)
	}

	fws.mu.Lock()
	fws.watcher = watcher
	fws.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			closeError := fws.watcher.Close()
			if closeError != nil {
				slog.ErrorContext(ctx, "Unable to close file watcher", "error", closeError)
			}

			return
		case event := <-fws.watcher.Events:
			fws.handleEvent(ctx, event)
		case <-instanceWatcherTicker.C:
			fws.checkForUpdates(ctx, ch)
		case watcherError := <-fws.watcher.Errors:
			slog.ErrorContext(ctx, "Unexpected error in file watcher", "error", watcherError)
		}
	}
}

func (fws *FileWatcherService) SetEnabled(enabled bool) {
	fws.enabled.Store(enabled)
}

func (fws *FileWatcherService) Update(ctx context.Context, nginxConfigContext *model.NginxConfigContext) {
	slog.DebugContext(ctx, "Updating file watcher", "nginx_config_context", nginxConfigContext)

	fws.mu.Lock()
	defer fws.mu.Unlock()

	directoriesToWatch := make(map[string]struct{})

	for _, file := range nginxConfigContext.Files {
		directoriesToWatch[filepath.Dir(file.GetFileMeta().GetName())] = struct{}{}
	}

	for _, file := range nginxConfigContext.Includes {
		directoriesToWatch[filepath.Dir(file)] = struct{}{}
	}

	fws.directoriesToWatch = directoriesToWatch

	if fws.watcher != nil {
		slog.InfoContext(ctx, "Updating file watcher", "allowed", fws.agentConfig.AllowedDirectories)

		// Start watching new directories
		fws.addWatchers(ctx)

		// Check if directories no longer need to be watched
		fws.removeWatchers(ctx)
	}
}

func (fws *FileWatcherService) addWatchers(ctx context.Context) {
	for directory := range fws.directoriesToWatch {
		if !fws.agentConfig.IsDirectoryAllowed(directory) {
			slog.WarnContext(
				ctx,
				"Unable to watch file in a directory that is not in allowed directory list",
				"directory", directory,
			)

			continue
		}

		if !slices.Contains(fws.watcher.WatchList(), directory) {
			err := fws.addWatcher(ctx, directory)
			if err != nil {
				slog.DebugContext(ctx, "Failed to add file watcher", "directory", directory, "error", err)
			} else {
				fws.filesChanged.Store(true)
			}
		}
	}
}

func (fws *FileWatcherService) removeWatchers(ctx context.Context) {
	for _, directoryBeingWatched := range fws.watcher.WatchList() {
		if _, err := os.Stat(directoryBeingWatched); errors.Is(err, os.ErrNotExist) {
			slog.DebugContext(
				ctx,
				"Directory does not exist removing watcher",
				"directory", directoryBeingWatched,
			)

			fws.removeWatcher(ctx, directoryBeingWatched)
			fws.filesChanged.Store(true)
		} else if _, ok := fws.directoriesToWatch[directoryBeingWatched]; !ok {
			fws.removeWatcher(ctx, directoryBeingWatched)
			fws.filesChanged.Store(true)
		}
	}
}

func (fws *FileWatcherService) handleEvent(ctx context.Context, event fsnotify.Event) {
	if fws.enabled.Load() {
		if fws.isEventSkippable(event) {
			return
		}

		slog.DebugContext(ctx, "Processing FSNotify event", "event", event)
		fws.filesChanged.Store(true)
	}
}

func (fws *FileWatcherService) checkForUpdates(ctx context.Context, ch chan<- FileUpdateMessage) {
	slog.DebugContext(ctx, "Checking for file watcher updates")

	fws.mu.Lock()
	defer fws.mu.Unlock()

	if fws.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create file watcher", "error", err)
			return
		}

		fws.watcher = watcher
	}

	// Start watching new directories
	fws.addWatchers(ctx)

	// Check if directories no longer need to be watched
	fws.removeWatchers(ctx)

	if fws.filesChanged.Load() && fws.enabled.Load() {
		newCtx := context.WithValue(
			ctx,
			logger.CorrelationIDContextKey,
			slog.Any(logger.CorrelationIDKey, logger.GenerateCorrelationID()),
		)

		slog.DebugContext(newCtx, "File watcher detected a file change")
		ch <- FileUpdateMessage{CorrelationID: logger.CorrelationIDAttr(newCtx)}
		fws.filesChanged.Store(false)
	}
}

func (fws *FileWatcherService) addWatcher(ctx context.Context, directory string) error {
	slog.DebugContext(ctx, "Checking if file watcher needs to be added", "directory", directory)

	if _, err := os.Stat(directory); errors.Is(err, os.ErrNotExist) {
		slog.DebugContext(
			ctx, "Unable to watch directory that does not exist",
			"directory", directory, "error", err,
		)
	}

	slog.DebugContext(ctx, "Adding watcher", "directory", directory)

	return fws.watcher.Add(directory)
}

func (fws *FileWatcherService) removeWatcher(ctx context.Context, path string) {
	slog.DebugContext(ctx, "Removing watcher", "directory", path)
	err := fws.watcher.Remove(path)
	if err != nil {
		slog.WarnContext(ctx, "Failed to remove file watcher", "directory_path", path, "error", err)
		return
	}
}

func (fws *FileWatcherService) isEventSkippable(event fsnotify.Event) bool {
	return event == emptyEvent ||
		event.Name == "" || isExcludedFile(event.Name, fws.agentConfig.Watchers.FileWatcher.ExcludeFiles)
}

func isExcludedFile(path string, excludeFiles []string) bool {
	path = strings.ToLower(path)
	for _, pattern := range excludeFiles {
		_, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			slog.Error("Invalid path for excluding file", "file_path", pattern)
			continue
		}

		ok, err := regexp.MatchString(pattern, path)
		if err != nil {
			slog.Error("Invalid path for excluding file", "file_path", pattern)
			continue
		} else if ok {
			return true
		}
	}

	return false
}
