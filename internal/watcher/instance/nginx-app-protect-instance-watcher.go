// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/id"
)

var (
	versionFilePath                = "/opt/app_protect/VERSION"
	releaseFilePath                = "/opt/app_protect/RELEASE"
	attackSignatureVersionFilePath = "/opt/app_protect/var/update_files/signatures/version"
	threatCampaignVersionFilePath  = "/opt/app_protect/var/update_files/threat_campaigns/version"
	enforcerEngineVersionFilePath  = "/opt/app_protect/bd_config/enforcer.version"

	versionFiles = []string{
		versionFilePath,
		releaseFilePath,
		attackSignatureVersionFilePath,
		threatCampaignVersionFilePath,
		enforcerEngineVersionFilePath,
	}
)

type NginxAppProtectInstanceWatcher struct {
	agentConfig             *config.Config
	watcher                 *fsnotify.Watcher
	instancesChannel        chan<- InstanceUpdatesMessage
	nginxAppProtectInstance *mpi.Instance
	filesBeingWatched       map[string]bool
	version                 string
	release                 string
	attackSignatureVersion  string
	threatCampaignVersion   string
	enforcerEngineVersion   string
}

func NewNginxAppProtectInstanceWatcher(agentConfig *config.Config) *NginxAppProtectInstanceWatcher {
	return &NginxAppProtectInstanceWatcher{
		agentConfig:       agentConfig,
		filesBeingWatched: make(map[string]bool),
	}
}

func (w *NginxAppProtectInstanceWatcher) Watch(ctx context.Context, instancesChannel chan<- InstanceUpdatesMessage) {
	monitoringFrequency := w.agentConfig.Watchers.InstanceWatcher.MonitoringFrequency
	slog.DebugContext(
		ctx,
		"Starting NGINX App Protect instance watcher monitoring",
		"monitoring_frequency", monitoringFrequency,
	)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create NGINX App Protect instance watcher", "error", err)
		return
	}

	w.watcher = watcher
	w.instancesChannel = instancesChannel

	w.watchVersionFiles(ctx)

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			closeError := w.watcher.Close()
			if closeError != nil {
				slog.ErrorContext(ctx, "Unable to close NGINX App Protect instance watcher", "error", closeError)
			}

			return
		case <-instanceWatcherTicker.C:
			// Need to keep watching directories in case NAP gets installed a while after NGINX Agent is started
			w.watchVersionFiles(ctx)
			w.checkForUpdates(ctx)
		case event := <-w.watcher.Events:
			w.handleEvent(ctx, event)
		case watcherError := <-w.watcher.Errors:
			slog.ErrorContext(ctx, "Unexpected error in NGINX App Protect instance watcher", "error", watcherError)
		}
	}
}

func (w *NginxAppProtectInstanceWatcher) watchVersionFiles(ctx context.Context) {
	for _, versionFile := range versionFiles {
		if !w.filesBeingWatched[versionFile] {
			if _, fileOs := os.Stat(versionFile); fileOs != nil && os.IsNotExist(fileOs) {
				w.filesBeingWatched[versionFile] = false
				continue
			}

			w.addWatcher(ctx, versionFile)
			w.filesBeingWatched[versionFile] = true

			// On startup we need to read the files initially if they are discovered for the first time
			w.readVersionFile(ctx, versionFile)
		}
	}
}

func (w *NginxAppProtectInstanceWatcher) addWatcher(ctx context.Context, versionFile string) {
	if err := w.watcher.Add(versionFile); err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to add NGINX App Protect file watcher",
			"file", versionFile, "error", err,
		)
		removeError := w.watcher.Remove(versionFile)
		if removeError != nil {
			slog.ErrorContext(
				ctx,
				"Failed to remove NGINX App Protect file watcher",
				"file", versionFile, "error", removeError,
			)
		}
	}

	slog.DebugContext(ctx, "Added NGINX App Protect file watcher", "file", versionFile)
}

func (w *NginxAppProtectInstanceWatcher) readVersionFile(ctx context.Context, versionFile string) {
	switch versionFile {
	case versionFilePath:
		w.version = w.readFile(ctx, versionFilePath)
	case releaseFilePath:
		w.release = w.readFile(ctx, releaseFilePath)
	case threatCampaignVersionFilePath:
		w.threatCampaignVersion = w.readFile(ctx, threatCampaignVersionFilePath)
	case enforcerEngineVersionFilePath:
		w.enforcerEngineVersion = w.readFile(ctx, enforcerEngineVersionFilePath)
	case attackSignatureVersionFilePath:
		w.attackSignatureVersion = w.readFile(ctx, attackSignatureVersionFilePath)
	}
}

func (w *NginxAppProtectInstanceWatcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Write), event.Has(fsnotify.Create):
		w.handleFileUpdateEvent(ctx, event)
	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		w.handleFileDeleteEvent(event)
	}
}

func (w *NginxAppProtectInstanceWatcher) handleFileUpdateEvent(ctx context.Context, event fsnotify.Event) {
	switch event.Name {
	case versionFilePath:
		w.version = w.readFile(ctx, event.Name)
	case releaseFilePath:
		w.release = w.readFile(ctx, event.Name)
	case attackSignatureVersionFilePath:
		w.attackSignatureVersion = w.readFile(ctx, event.Name)
	case enforcerEngineVersionFilePath:
		w.enforcerEngineVersion = w.readFile(ctx, event.Name)
	case threatCampaignVersionFilePath:
		w.threatCampaignVersion = w.readFile(ctx, event.Name)
	}
}

func (w *NginxAppProtectInstanceWatcher) handleFileDeleteEvent(event fsnotify.Event) {
	switch event.Name {
	case versionFilePath:
		w.version = ""
	case releaseFilePath:
		w.release = ""
	case attackSignatureVersionFilePath:
		w.attackSignatureVersion = ""
	case enforcerEngineVersionFilePath:
		w.enforcerEngineVersion = ""
	case threatCampaignVersionFilePath:
		w.threatCampaignVersion = ""
	}
}

func (w *NginxAppProtectInstanceWatcher) checkForUpdates(ctx context.Context) {
	// If a version file is discovered for the first time we treat that as a new instance
	if w.isNewInstance() {
		w.createInstance(ctx)
	} else if w.nginxAppProtectInstance != nil {
		// If a version file disappears then we assume that NGINX App Protect is uninstalled
		if w.version == "" {
			w.deleteInstance(ctx)
			// If any version changes then we update the instance metadata
		} else if w.haveVersionsChanged() {
			w.updateInstance(ctx)
		}
	}
}

func (w *NginxAppProtectInstanceWatcher) isNewInstance() bool {
	return w.nginxAppProtectInstance == nil && w.version != ""
}

func (w *NginxAppProtectInstanceWatcher) createInstance(ctx context.Context) {
	w.nginxAppProtectInstance = &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   id.Generate(versionFilePath),
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX_APP_PROTECT,
			Version:      w.version,
		},
		InstanceConfig: &mpi.InstanceConfig{},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  0,
			BinaryPath: "",
			ConfigPath: "",
			Details: &mpi.InstanceRuntime_NginxAppProtectRuntimeInfo{
				NginxAppProtectRuntimeInfo: &mpi.NGINXAppProtectRuntimeInfo{
					Release:                w.release,
					AttackSignatureVersion: w.attackSignatureVersion,
					ThreatCampaignVersion:  w.threatCampaignVersion,
					EnforcerEngineVersion:  w.enforcerEngineVersion,
				},
			},
			InstanceChildren: make([]*mpi.InstanceChild, 0),
		},
	}

	slog.InfoContext(ctx, "Discovered a new NGINX App Protect instance")

	w.instancesChannel <- InstanceUpdatesMessage{
		CorrelationID: logger.CorrelationIDAttr(ctx),
		InstanceUpdates: InstanceUpdates{
			NewInstances: []*mpi.Instance{
				w.nginxAppProtectInstance,
			},
		},
	}
}

func (w *NginxAppProtectInstanceWatcher) deleteInstance(ctx context.Context) {
	slog.InfoContext(ctx, "NGINX App Protect instance not longer exists")

	w.instancesChannel <- InstanceUpdatesMessage{
		CorrelationID: logger.CorrelationIDAttr(ctx),
		InstanceUpdates: InstanceUpdates{
			DeletedInstances: []*mpi.Instance{
				w.nginxAppProtectInstance,
			},
		},
	}
	w.nginxAppProtectInstance = nil
}

func (w *NginxAppProtectInstanceWatcher) updateInstance(ctx context.Context) {
	w.nginxAppProtectInstance.GetInstanceMeta().Version = w.version
	runtimeInfo := w.nginxAppProtectInstance.GetInstanceRuntime().GetNginxAppProtectRuntimeInfo()
	runtimeInfo.Release = w.release
	runtimeInfo.AttackSignatureVersion = w.attackSignatureVersion
	runtimeInfo.ThreatCampaignVersion = w.threatCampaignVersion
	runtimeInfo.EnforcerEngineVersion = w.enforcerEngineVersion

	slog.DebugContext(ctx, "NGINX App Protect instance updated")

	w.instancesChannel <- InstanceUpdatesMessage{
		CorrelationID: logger.CorrelationIDAttr(ctx),
		InstanceUpdates: InstanceUpdates{
			UpdatedInstances: []*mpi.Instance{
				w.nginxAppProtectInstance,
			},
		},
	}
}

func (w *NginxAppProtectInstanceWatcher) haveVersionsChanged() bool {
	version := w.nginxAppProtectInstance.GetInstanceMeta().GetVersion()
	runtimeInfo := w.nginxAppProtectInstance.GetInstanceRuntime().GetNginxAppProtectRuntimeInfo()

	return version != w.version ||
		runtimeInfo.GetRelease() != w.release ||
		runtimeInfo.GetAttackSignatureVersion() != w.attackSignatureVersion ||
		runtimeInfo.GetThreatCampaignVersion() != w.threatCampaignVersion ||
		runtimeInfo.GetEnforcerEngineVersion() != w.enforcerEngineVersion
}

func (w *NginxAppProtectInstanceWatcher) readFile(ctx context.Context, filePath string) string {
	contents, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		slog.DebugContext(ctx, "Unable to read NGINX App Protect file", "file_path", filePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(contents), "\n")
}
