// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package credentials

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
)

const (
	monitoringInterval = 5 * time.Second
)

var emptyEvent = fsnotify.Event{
	Name: "",
	Op:   0,
}

type CredentialUpdateMessage struct {
	CorrelationID slog.Attr
}

type CredentialWatcherService struct {
	enabled           *atomic.Bool
	agentConfig       *config.Config
	watcher           *fsnotify.Watcher
	filesBeingWatched *sync.Map
	filesChanged      *atomic.Bool
}

func NewCredentialWatcherService(agentConfig *config.Config) *CredentialWatcherService {
	enabled := &atomic.Bool{}
	enabled.Store(true)

	filesChanged := &atomic.Bool{}
	filesChanged.Store(false)

	return &CredentialWatcherService{
		enabled:           enabled,
		agentConfig:       agentConfig,
		filesBeingWatched: &sync.Map{},
		filesChanged:      filesChanged,
	}
}

func (cws *CredentialWatcherService) Watch(ctx context.Context, ch chan<- CredentialUpdateMessage) {
	slog.DebugContext(ctx, "Starting credential watcher monitoring")

	ticker := time.NewTicker(monitoringInterval)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create credential watcher", "error", err)
		return
	}

	cws.watcher = watcher

	cws.watchFiles(ctx, credentialPaths(cws.agentConfig))

	for {
		select {
		case <-ctx.Done():
			closeError := cws.watcher.Close()
			if closeError != nil {
				slog.ErrorContext(ctx, "Unable to close credential watcher", "error", closeError)
			}

			return
		case event := <-cws.watcher.Events:
			cws.handleEvent(ctx, event)
		case <-ticker.C:
			cws.checkForUpdates(ctx, ch)
		case watcherError := <-cws.watcher.Errors:
			slog.ErrorContext(ctx, "Unexpected error in credential watcher", "error", watcherError)
		}
	}
}

func (cws *CredentialWatcherService) SetEnabled(enabled bool) {
	cws.enabled.Store(enabled)
}

func (cws *CredentialWatcherService) addWatcher(ctx context.Context, filePath string) {
	if !cws.enabled.Load() {
		slog.DebugContext(ctx, "Credential watcher is disabled")

		return
	}

	if cws.isWatching(filePath) {
		slog.DebugContext(
			ctx, "Credential watcher is already watching ", "path", filePath)

		return
	}

	if err := cws.watcher.Add(filePath); err != nil {
		slog.ErrorContext(ctx, "Failed to add credential watcher", "path", filePath, "error", err)
		removeError := cws.watcher.Remove(filePath)
		if removeError != nil {
			slog.ErrorContext(
				ctx, "Failed to remove credential watcher", "path", filePath, "error", removeError)
		}

		return
	}
	cws.filesBeingWatched.Store(filePath, true)
	slog.DebugContext(ctx, "Credential watcher has been added", "path", filePath)
}

func (cws *CredentialWatcherService) watchFiles(ctx context.Context, files []string) {
	slog.DebugContext(ctx, "Creating credential watchers")

	for _, filePath := range files {
		cws.addWatcher(ctx, filePath)
	}
}

func (cws *CredentialWatcherService) isWatching(path string) bool {
	v, _ := cws.filesBeingWatched.LoadOrStore(path, false)

	if value, ok := v.(bool); ok {
		return value
	}

	return false
}

func (cws *CredentialWatcherService) handleEvent(ctx context.Context, event fsnotify.Event) {
	if cws.enabled.Load() {
		if isEventSkippable(event) {
			slog.DebugContext(ctx, "Skipping FSNotify event", "event", event)
			return
		}

		slog.DebugContext(ctx, "Processing FSNotify event", "event", event)

		switch {
		case event.Has(fsnotify.Rename):
			if !slices.Contains(cws.watcher.WatchList(), event.Name) {
				cws.filesBeingWatched.Store(event.Name, false)
			}
			cws.addWatcher(ctx, event.Name)
		}

		cws.filesChanged.Store(true)
	}
}

func (cws *CredentialWatcherService) checkForUpdates(ctx context.Context, ch chan<- CredentialUpdateMessage) {
	if cws.filesChanged.Load() {
		newCtx := context.WithValue(
			ctx,
			logger.CorrelationIDContextKey,
			slog.Any(logger.CorrelationIDKey, logger.GenerateCorrelationID()),
		)

		slog.DebugContext(ctx, "Credential watcher has detected changes")
		ch <- CredentialUpdateMessage{CorrelationID: logger.GetCorrelationIDAttr(newCtx)}
		cws.filesChanged.Store(false)
	}
}

func credentialPaths(agentConfig *config.Config) []string {
	var paths []string

	if agentConfig.Command.Auth != nil {
		if agentConfig.Command.Auth.TokenPath != "" {
			paths = append(paths, agentConfig.Command.Auth.TokenPath)
		}
	}

	return paths
}

func isEventSkippable(event fsnotify.Event) bool {
	return event == emptyEvent ||
		event.Name == "" ||
		event.Has(fsnotify.Chmod) ||
		event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Remove)
}
