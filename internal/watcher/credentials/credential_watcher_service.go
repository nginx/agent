package credentials

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/nginx/agent/v3/internal/config"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
		//case <-instanceWatcherTicker.C:
		//	cws.checkForUpdates(ctx, ch)
		case watcherError := <-cws.watcher.Errors:
			slog.ErrorContext(ctx, "Unexpected error in credential watcher", "error", watcherError)
		}
	}
}

func (cws *CredentialWatcherService) SetEnabled(enabled bool) {
	cws.enabled.Store(enabled)
}

func (cws *CredentialWatcherService) addWatcher(ctx context.Context, filePath string, info os.FileInfo) {

	if !cws.enabled.Load() {
		slog.DebugContext(ctx, "Credential watcher is disabled")

		return
	}

	if cws.isWatching(filePath) {
		slog.DebugContext(
			ctx, "Credential watcher is already watching ", "path", filePath)

		return
	}

	if !info.IsDir() && !cws.isWatching(filePath) {
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
}

func (cws *CredentialWatcherService) removeWatcher(ctx context.Context, filePath string) {
	if _, ok := cws.filesBeingWatched.Load(filePath); ok {
		err := cws.watcher.Remove(filePath)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to remove credential watcher", "path", filePath, "error", err)
			return
		}

		cws.filesBeingWatched.Delete(filePath)
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

		switch {
		case event.Op&Write == Write:
			// We want to send messages on write since that means the contents changed,
			// but we also need to restart the gRPC connection to load the new credentials from the file being watched.
			// Can we post a message on the message bus to trigger the CommandService to restart the gRPC connection?
		case event.Op&Create == Create:
			//info, err := os.Stat(event.Name)
			//if err != nil {
			//	slog.DebugContext(ctx, "Unable to add watcher", "path", event.Name, "error", err)
			//	return
			//}
			//cws.addWatcher(ctx, event.Name, info)
		case event.Op&Remove == Remove, event.Op&Rename == Rename:
			//cws.removeWatcher(ctx, event.Name)
		}

		slog.DebugContext(ctx, "Processing FSNotify event", "event", event)

		cws.filesChanged.Store(true)
	}
}

func isEventSkippable(event fsnotify.Event) bool {
	return event == emptyEvent ||
		event.Name == "" ||
		strings.HasSuffix(event.Name, ".swp") ||
		strings.HasSuffix(event.Name, ".swx") ||
		strings.HasSuffix(event.Name, "~")
}
