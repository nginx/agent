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

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/grpc"

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
	CorrelationID  slog.Attr
	GrpcConnection *grpc.GrpcConnection
	ServerType     model.ServerType
}

type CredentialWatcherService struct {
	agentConfig       *config.Config
	watcher           *fsnotify.Watcher
	filesBeingWatched *sync.Map
	filesChanged      *atomic.Bool
	serverType        model.ServerType
	watcherMutex      sync.Mutex
}

func NewCredentialWatcherService(agentConfig *config.Config, serverType model.ServerType) *CredentialWatcherService {
	filesChanged := &atomic.Bool{}
	filesChanged.Store(false)

	return &CredentialWatcherService{
		agentConfig:       agentConfig,
		filesBeingWatched: &sync.Map{},
		filesChanged:      filesChanged,
		serverType:        serverType,
		watcherMutex:      sync.Mutex{},
	}
}

func (cws *CredentialWatcherService) Watch(ctx context.Context, ch chan<- CredentialUpdateMessage) {
	newCtx := context.WithValue(
		ctx,
		logger.ServerTypeContextKey,
		slog.Any(logger.ServerTypeKey, cws.serverType.String()),
	)
	slog.DebugContext(newCtx, "Starting credential watcher monitoring")

	ticker := time.NewTicker(monitoringInterval)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(newCtx, "Failed to create credential watcher", "error", err)
		return
	}

	cws.watcher = watcher

	cws.watcherMutex.Lock()
	commandServer := cws.agentConfig.Command

	if cws.serverType == model.Auxiliary {
		commandServer = cws.agentConfig.AuxiliaryCommand
	}

	cws.watchFiles(newCtx, credentialPaths(commandServer))
	cws.watcherMutex.Unlock()

	for {
		select {
		case <-newCtx.Done():
			closeError := cws.watcher.Close()
			if closeError != nil {
				slog.ErrorContext(newCtx, "Unable to close credential watcher", "error", closeError)
			}

			return
		case event := <-cws.watcher.Events:
			cws.handleEvent(newCtx, event)
		case <-ticker.C:
			cws.checkForUpdates(newCtx, ch)
		case watcherError := <-cws.watcher.Errors:
			slog.ErrorContext(newCtx, "Unexpected error in credential watcher", "error", watcherError)
		}
	}
}

func (cws *CredentialWatcherService) addWatcher(ctx context.Context, filePath string) {
	if cws.isWatching(filePath) {
		slog.DebugContext(
			ctx, "Credential watcher is already watching ", "path", filePath)

		return
	}

	if err := cws.watcher.Add(filePath); err != nil {
		slog.ErrorContext(ctx, "Failed to add credential watcher", "path", filePath, "error", err)

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
	if isEventSkippable(event) {
		slog.DebugContext(ctx, "Skipping FSNotify event", "event", event)
		return
	}

	slog.DebugContext(ctx, "Processing FSNotify event", "event", event)

	switch {
	case event.Has(fsnotify.Remove):
		fallthrough
	case event.Has(fsnotify.Rename):
		if !slices.Contains(cws.watcher.WatchList(), event.Name) {
			cws.filesBeingWatched.Store(event.Name, false)
		}
		cws.addWatcher(ctx, event.Name)
	}

	cws.filesChanged.Store(true)
}

func (cws *CredentialWatcherService) checkForUpdates(ctx context.Context, ch chan<- CredentialUpdateMessage) {
	if cws.filesChanged.Load() {
		newCtx := context.WithValue(
			ctx,
			logger.CorrelationIDContextKey,
			slog.Any(logger.CorrelationIDKey, logger.GenerateCorrelationID()),
		)

		cws.watcherMutex.Lock()
		defer cws.watcherMutex.Unlock()

		commandServer := cws.agentConfig.Command
		if cws.serverType == model.Auxiliary {
			commandServer = cws.agentConfig.AuxiliaryCommand
		}

		conn, err := grpc.NewGrpcConnection(newCtx, cws.agentConfig, commandServer)
		if err != nil {
			slog.ErrorContext(newCtx, "Unable to create new grpc connection", "error", err)
			cws.filesChanged.Store(false)

			return
		}
		slog.DebugContext(ctx, "Credential watcher has detected changes")
		ch <- CredentialUpdateMessage{
			CorrelationID:  logger.CorrelationIDAttr(newCtx),
			ServerType:     cws.serverType,
			GrpcConnection: conn,
		}
		cws.filesChanged.Store(false)
	}
}

func credentialPaths(agentConfig *config.Command) []string {
	var paths []string

	if agentConfig.Auth != nil {
		if agentConfig.Auth.TokenPath != "" {
			paths = append(paths, agentConfig.Auth.TokenPath)
		}
	}

	// agent's tls certs
	if agentConfig.TLS != nil {
		if agentConfig.TLS.Ca != "" {
			paths = append(paths, agentConfig.TLS.Ca)
		}
		if agentConfig.TLS.Cert != "" {
			paths = append(paths, agentConfig.TLS.Cert)
		}
		if agentConfig.TLS.Key != "" {
			paths = append(paths, agentConfig.TLS.Key)
		}
	}

	return paths
}

func isEventSkippable(event fsnotify.Event) bool {
	return event == emptyEvent ||
		event.Name == "" ||
		event.Has(fsnotify.Chmod) ||
		event.Has(fsnotify.Create)
}
