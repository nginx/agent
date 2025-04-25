// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
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

var logOrigin = slog.String("log_origin", "file_watcher_service.go")

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
	slog.DebugContext(
		ctx,
		"Starting file watcher monitoring",
		"monitoring_frequency", monitoringFrequency,
		logOrigin,
	)

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to create file watcher",
			"error", err,
			logOrigin,
		)

		return
	}

	fws.watcher = watcher

	fws.watchDirectories(ctx)

	for {
		select {
		case <-ctx.Done():
			closeError := fws.watcher.Close()
			if closeError != nil {
				slog.ErrorContext(ctx, "Unable to close file watcher", "error", closeError, logOrigin)
			}

			return
		case event := <-fws.watcher.Events:
			fws.handleEvent(ctx, event)
		case <-instanceWatcherTicker.C:
			fws.checkForUpdates(ctx, ch)
		case watcherError := <-fws.watcher.Errors:
			slog.ErrorContext(ctx, "Unexpected error in file watcher", "error", watcherError, logOrigin)
		}
	}
}

func (fws *FileWatcherService) SetEnabled(enabled bool) {
	fws.enabled.Store(enabled)
}

func (fws *FileWatcherService) watchDirectories(ctx context.Context) {
	for _, dir := range fws.agentConfig.AllowedDirectories {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			slog.DebugContext(
				ctx,
				"Unable to watch directory that does not exist",
				"directory", dir,
				"error", err,
				logOrigin,
			)

			continue
		}

		slog.DebugContext(ctx, "Creating file watchers", "directory", dir, logOrigin)

		err := fws.walkDir(ctx, dir)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"Failed to create file watchers",
				"directory", dir,
				"error", err,
				logOrigin,
			)
		}
	}
}

func (fws *FileWatcherService) walkDir(ctx context.Context, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, fileWalkErr error) error {
		if fileWalkErr != nil {
			return fileWalkErr
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			slog.ErrorContext(
				ctx,
				"Error getting info for file",
				"error", infoErr,
				logOrigin,
			)

			return infoErr
		}

		if d.IsDir() {
			fws.addWatcher(ctx, path, info)
		}

		return nil
	})
}

func (fws *FileWatcherService) addWatcher(ctx context.Context, path string, info os.FileInfo) {
	if info.IsDir() && !fws.isWatching(path) {
		if err := fws.watcher.Add(path); err != nil {
			slog.ErrorContext(
				ctx,
				"Failed to add file watcher",
				"directory_path", path,
				"error", err,
				logOrigin,
			)
			removeError := fws.watcher.Remove(path)
			if removeError != nil {
				slog.ErrorContext(
					ctx,
					"Failed to remove file watcher",
					"directory_path", path,
					"error", removeError,
					logOrigin,
				)
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
			slog.ErrorContext(
				ctx,
				"Failed to remove file watcher",
				"directory_path", path,
				"error", err,
				logOrigin,
			)

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
		if fws.isEventSkippable(event) {
			return
		}

		switch {
		case event.Op&Write == Write:
			// We want to send messages on write since that means the contents changed,
			// but we already have a watcher on the file so nothing special needs to happen here
		case event.Op&Create == Create:
			info, err := os.Stat(event.Name)
			if err != nil {
				slog.DebugContext(ctx, "Unable to add watcher", "path", event.Name, "error", err, logOrigin)
				return
			}
			fws.addWatcher(ctx, event.Name, info)
		case event.Op&Remove == Remove, event.Op&Rename == Rename:
			fws.removeWatcher(ctx, event.Name)
		}

		slog.DebugContext(ctx, "Processing FSNotify event", "event", event, logOrigin)

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

		slog.DebugContext(newCtx, "File watcher detected a file change", logOrigin)
		ch <- FileUpdateMessage{CorrelationID: logger.GetCorrelationIDAttr(newCtx)}
		fws.filesChanged.Store(false)
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
			slog.Error("Invalid path for excluding file", "file_path", pattern, logOrigin)
			continue
		}

		ok, err := regexp.MatchString(pattern, path)
		if err != nil {
			slog.Error("Invalid path for excluding file", "file_path", pattern, logOrigin)
			continue
		} else if ok {
			return true
		}
	}

	return false
}
