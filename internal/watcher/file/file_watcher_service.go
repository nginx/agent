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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
)

const (
	Create = fsnotify.Create
	Write  = fsnotify.Write
	Remove = fsnotify.Remove
	Rename = fsnotify.Rename
	Chmod  = fsnotify.Chmod
)

var emptyEvent = fsnotify.Event{
	Name: "",
	Op:   0,
}

type FileUpdateMessage struct {
	CorrelationID slog.Attr
}

type FileWatcherService struct {
	agentConfig             *config.Config
	watcher                 *fsnotify.Watcher
	directoriesBeingWatched *sync.Map
	filesChanged            *atomic.Bool
	enabled                 *atomic.Bool
}

func NewFileWatcherService(agentConfig *config.Config) *FileWatcherService {
	enabled := &atomic.Bool{}
	enabled.Store(true)

	filesChanged := &atomic.Bool{}
	filesChanged.Store(false)

	return &FileWatcherService{
		agentConfig:             agentConfig,
		directoriesBeingWatched: &sync.Map{},
		enabled:                 enabled,
		filesChanged:            filesChanged,
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
		return
	}

	fws.watcher = watcher

	fws.watchDirectories(ctx)

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

func (fws *FileWatcherService) watchDirectories(ctx context.Context) {
	for _, dir := range fws.agentConfig.AllowedDirectories {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			slog.DebugContext(ctx, "Unable to watch directory that does not exist", "directory", dir, "error", err)
			continue
		}

		slog.DebugContext(ctx, "Creating file watchers", "directory", dir)

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				fws.addWatcher(ctx, path, info)
			}

			return nil
		})
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create file watchers", "directory", dir, "error", err)
		}
	}
}

func (fws *FileWatcherService) addWatcher(ctx context.Context, path string, info os.FileInfo) {
	if info.IsDir() && !fws.isWatching(path) {
		if err := fws.watcher.Add(path); err != nil {
			slog.ErrorContext(ctx, "Failed to add file watcher", "directory_path", path, "error", err)
			removeError := fws.watcher.Remove(path)
			if removeError != nil {
				slog.ErrorContext(ctx, "Failed to remove file watcher", "directory_path", path, "error", removeError)
			}

			return
		}

		fws.directoriesBeingWatched.Store(path, true)
	}
}

func (fws *FileWatcherService) removeWatcher(ctx context.Context, path string) {
	if _, ok := fws.directoriesBeingWatched.Load(path); ok {
		err := fws.watcher.Remove(path)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to remove file watcher", "directory_path", path, "error", err)
			return
		}

		fws.directoriesBeingWatched.Delete(path)
	}
}

func (fws *FileWatcherService) isWatching(name string) bool {
	v, _ := fws.directoriesBeingWatched.LoadOrStore(name, false)

	if value, ok := v.(bool); ok {
		return value
	}

	return false
}

func (fws *FileWatcherService) handleEvent(ctx context.Context, event fsnotify.Event) {
	if fws.enabled.Load() {
		if isEventSkippable(event) {
			slog.DebugContext(ctx, "Skipping FSNotify event", "event", event)
			return
		}

		switch {
		case event.Op&Write == Write:
			// We want to send messages on write since that means the contents changed,
			// but we already have a watcher on the file so nothing special needs to happen here
		case event.Op&Create == Create:
			info, err := os.Stat(event.Name)
			if err != nil {
				slog.DebugContext(ctx, "Unable to add watcher", "path", event.Name, "error", err)
				return
			}
			fws.addWatcher(ctx, event.Name, info)
		case event.Op&Remove == Remove, event.Op&Rename == Rename:
			fws.removeWatcher(ctx, event.Name)
		}

		slog.DebugContext(ctx, "Processing FSNotify event", "event", event)

		fws.filesChanged.Store(true)
	}
}

func (fws *FileWatcherService) checkForUpdates(ctx context.Context, ch chan<- FileUpdateMessage) {
	if fws.filesChanged.Load() {
		newCtx := context.WithValue(
			ctx,
			logger.CorrelationIDContextKey,
			slog.Any(logger.CorrelationIDKey, logger.GenerateCorrelationID()),
		)

		slog.DebugContext(newCtx, "File watcher detected a file change")
		ch <- FileUpdateMessage{CorrelationID: logger.GetCorrelationIDAttr(newCtx)}
		fws.filesChanged.Store(false)
	}
}

func isEventSkippable(event fsnotify.Event) bool {
	return event == emptyEvent ||
		event.Name == "" ||
		strings.HasSuffix(event.Name, ".swp") ||
		strings.HasSuffix(event.Name, ".swx") ||
		strings.HasSuffix(event.Name, "~")
}
