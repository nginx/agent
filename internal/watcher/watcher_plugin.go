// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/watcher/credentials"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/internal/watcher/file"
	"github.com/nginx/agent/v3/internal/watcher/health"
	"github.com/nginx/agent/v3/internal/watcher/instance"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	pkgConfig "github.com/nginx/agent/v3/pkg/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . instanceWatcherServiceInterface

// nolint
type (
	Watcher struct {
		messagePipe                        bus.MessagePipeInterface
		agentConfig                        *config.Config
		instanceWatcherService             instanceWatcherServiceInterface
		nginxAppProtectInstanceWatcher     *instance.NginxAppProtectInstanceWatcher
		healthWatcherService               *health.HealthWatcherService
		fileWatcherService                 *file.FileWatcherService
		commandCredentialWatcherService    credentialWatcherServiceInterface
		auxiliaryCredentialWatcherService  credentialWatcherServiceInterface
		instanceUpdatesChannel             chan instance.InstanceUpdatesMessage
		nginxConfigContextChannel          chan instance.NginxConfigContextMessage
		instanceHealthChannel              chan health.InstanceHealthMessage
		fileUpdatesChannel                 chan file.FileUpdateMessage
		commandCredentialUpdatesChannel    chan credentials.CredentialUpdateMessage
		auxiliaryCredentialUpdatesChannel  chan credentials.CredentialUpdateMessage
		cancel                             context.CancelFunc
		instancesWithConfigApplyInProgress []string
		watcherMutex                       sync.Mutex
	}

	instanceWatcherServiceInterface interface {
		Watch(
			ctx context.Context,
			instancesChannel chan<- instance.InstanceUpdatesMessage,
			nginxConfigContextChannel chan<- instance.NginxConfigContextMessage,
		)
		HandleNginxConfigContextUpdate(ctx context.Context, instanceID string, configContext *model.NginxConfigContext)
		ReparseConfigs(ctx context.Context)
		SetEnabled(enabled bool)
	}

	credentialWatcherServiceInterface interface {
		Watch(
			ctx context.Context,
			credentialUpdateChannel chan<- credentials.CredentialUpdateMessage,
		)
	}
)

var _ bus.Plugin = (*Watcher)(nil)

func NewWatcher(agentConfig *config.Config) *Watcher {
	return &Watcher{
		agentConfig:                        agentConfig,
		instanceWatcherService:             instance.NewInstanceWatcherService(agentConfig),
		nginxAppProtectInstanceWatcher:     instance.NewNginxAppProtectInstanceWatcher(agentConfig),
		healthWatcherService:               health.NewHealthWatcherService(agentConfig),
		fileWatcherService:                 file.NewFileWatcherService(agentConfig),
		commandCredentialWatcherService:    credentials.NewCredentialWatcherService(agentConfig, model.Command),
		auxiliaryCredentialWatcherService:  credentials.NewCredentialWatcherService(agentConfig, model.Auxiliary),
		instanceUpdatesChannel:             make(chan instance.InstanceUpdatesMessage),
		nginxConfigContextChannel:          make(chan instance.NginxConfigContextMessage),
		instanceHealthChannel:              make(chan health.InstanceHealthMessage),
		fileUpdatesChannel:                 make(chan file.FileUpdateMessage),
		commandCredentialUpdatesChannel:    make(chan credentials.CredentialUpdateMessage),
		auxiliaryCredentialUpdatesChannel:  make(chan credentials.CredentialUpdateMessage),
		instancesWithConfigApplyInProgress: []string{},
		watcherMutex:                       sync.Mutex{},
	}
}

// nolint: unparam
// error is always nil
func (w *Watcher) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting watcher plugin")
	w.messagePipe = messagePipe

	watcherContext, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go w.nginxAppProtectInstanceWatcher.Watch(watcherContext, w.instanceUpdatesChannel)
	go w.instanceWatcherService.Watch(watcherContext, w.instanceUpdatesChannel, w.nginxConfigContextChannel)
	go w.healthWatcherService.Watch(watcherContext, w.instanceHealthChannel)
	go w.commandCredentialWatcherService.Watch(watcherContext, w.commandCredentialUpdatesChannel)

	if w.agentConfig.AuxiliaryCommand != nil {
		go w.auxiliaryCredentialWatcherService.Watch(watcherContext, w.auxiliaryCredentialUpdatesChannel)
	}

	if w.agentConfig.IsFeatureEnabled(pkgConfig.FeatureFileWatcher) {
		go w.fileWatcherService.Watch(watcherContext, w.fileUpdatesChannel)
	} else {
		slog.DebugContext(watcherContext, "File watcher feature is disabled",
			"enabled_features", w.agentConfig.Features)
	}

	go w.monitorWatchers(watcherContext)

	return nil
}

// nolint: unparam
// error is always nil
func (w *Watcher) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing watcher plugin")

	w.cancel()

	return nil
}

func (*Watcher) Info() *bus.Info {
	return &bus.Info{
		Name: "watcher",
	}
}

func (w *Watcher) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ConfigApplyRequestTopic:
		w.handleConfigApplyRequest(ctx, msg)
	case bus.ConfigApplySuccessfulTopic:
		w.handleConfigApplySuccess(ctx, msg)
	case bus.ConfigApplyCompleteTopic:
		w.handleConfigApplyComplete(ctx, msg)
	case bus.DataPlaneHealthRequestTopic:
		w.handleHealthRequest(ctx)
	default:
		slog.DebugContext(ctx, "Watcher plugin unknown topic", "topic", msg.Topic)
	}
}

func (*Watcher) Subscriptions() []string {
	return []string{
		bus.ConfigApplyRequestTopic,
		bus.ConfigApplySuccessfulTopic,
		bus.ConfigApplyCompleteTopic,
		bus.DataPlaneHealthRequestTopic,
	}
}

func (w *Watcher) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Watcher plugin received config apply request message")
	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest",
			"payload", msg.Data, "topic", msg.Topic)

		return
	}

	request, requestOk := managementPlaneRequest.GetRequest().(*mpi.ManagementPlaneRequest_ConfigApplyRequest)
	if !requestOk {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ConfigApplyRequest",
			"payload", msg.Data, "topic", msg.Topic)

		return
	}

	instanceID := request.ConfigApplyRequest.GetOverview().GetConfigVersion().GetInstanceId()

	w.watcherMutex.Lock()
	defer w.watcherMutex.Unlock()
	w.instancesWithConfigApplyInProgress = append(w.instancesWithConfigApplyInProgress, instanceID)

	w.fileWatcherService.SetEnabled(false)
	w.instanceWatcherService.SetEnabled(false)
}

func (w *Watcher) handleConfigApplySuccess(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Watcher plugin received config apply success message")
	successMessage, ok := msg.Data.(*model.ConfigApplySuccess)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplySuccess", "payload",
			msg.Data, "topic", msg.Topic)

		return
	}

	instanceID := successMessage.DataPlaneResponse.GetInstanceId()

	// If the config apply had no changes to any files, it is results in a ConfigApplySuccessfulTopic with an empty
	// configContext being sent, there is no need to reparse the config as no change has occurred.
	if successMessage.ConfigContext.InstanceID != "" {
		w.instanceWatcherService.HandleNginxConfigContextUpdate(ctx, instanceID, successMessage.ConfigContext)
	}

	w.watcherMutex.Lock()
	w.instancesWithConfigApplyInProgress = slices.DeleteFunc(
		w.instancesWithConfigApplyInProgress,
		func(element string) bool {
			return element == instanceID
		},
	)

	w.fileWatcherService.SetEnabled(true)
	w.instanceWatcherService.SetEnabled(true)
	w.watcherMutex.Unlock()
}

func (w *Watcher) handleHealthRequest(ctx context.Context) {
	slog.DebugContext(ctx, "Watcher plugin received health request message")
	w.messagePipe.Process(ctx, &bus.Message{
		Topic: bus.DataPlaneHealthResponseTopic, Data: w.healthWatcherService.InstancesHealth(),
	})
}

func (w *Watcher) handleConfigApplyComplete(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Watcher plugin received config apply complete message")
	response, ok := msg.Data.(*mpi.DataPlaneResponse)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.DataPlaneResponse", "payload",
			msg.Data, "topic", msg.Topic)

		return
	}

	instanceID := response.GetInstanceId()

	w.watcherMutex.Lock()
	defer w.watcherMutex.Unlock()
	w.instancesWithConfigApplyInProgress = slices.DeleteFunc(
		w.instancesWithConfigApplyInProgress,
		func(element string) bool {
			return element == instanceID
		},
	)

	w.instanceWatcherService.SetEnabled(true)
	w.fileWatcherService.SetEnabled(true)
}

func (w *Watcher) monitorWatchers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-w.commandCredentialUpdatesChannel:
			w.handleCredentialUpdate(ctx, message)
		case message := <-w.auxiliaryCredentialUpdatesChannel:
			w.handleCredentialUpdate(ctx, message)
		case message := <-w.instanceUpdatesChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			w.handleInstanceUpdates(newCtx, message)
		case message := <-w.nginxConfigContextChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			w.watcherMutex.Lock()

			if !slices.Contains(w.instancesWithConfigApplyInProgress, message.NginxConfigContext.InstanceID) {
				slog.DebugContext(
					newCtx,
					"Updated NGINX config context",
					"nginx_config_context", message.NginxConfigContext,
				)
				w.messagePipe.Process(
					newCtx,
					&bus.Message{Topic: bus.
						NginxConfigUpdateTopic, Data: message.NginxConfigContext},
				)
			} else {
				slog.DebugContext(
					newCtx,
					"Not sending updated NGINX config context since config apply is in progress",
					"nginx_config_context", message.NginxConfigContext,
				)
			}

			w.fileWatcherService.Update(ctx, message.NginxConfigContext)

			w.watcherMutex.Unlock()
		case message := <-w.instanceHealthChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			w.messagePipe.Process(newCtx, &bus.Message{
				Topic: bus.InstanceHealthTopic, Data: message.InstanceHealth,
			})
		case message := <-w.fileUpdatesChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			// Running this in a separate go routine otherwise we get into a deadlock
			// since the ReparseConfigs function could add new messages to one of the other watcher channels
			go w.instanceWatcherService.ReparseConfigs(newCtx)
		}
	}
}

func (w *Watcher) handleCredentialUpdate(ctx context.Context, message credentials.CredentialUpdateMessage) {
	newCtx := context.WithValue(context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID),
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey,
			message.ServerType.String()))

	slog.DebugContext(newCtx, "Received credential update event for command server")
	w.messagePipe.Process(newCtx, &bus.Message{
		Topic: bus.ConnectionResetTopic, Data: message.GrpcConnection,
	})
}

func (w *Watcher) handleInstanceUpdates(newCtx context.Context, message instance.InstanceUpdatesMessage) {
	if len(message.InstanceUpdates.NewInstances) > 0 {
		slog.DebugContext(newCtx, "New instances found", "instances", message.InstanceUpdates.NewInstances)
		w.healthWatcherService.AddHealthWatcher(message.InstanceUpdates.NewInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.AddInstancesTopic, Data: message.InstanceUpdates.NewInstances},
		)
	}
	if len(message.InstanceUpdates.UpdatedInstances) > 0 {
		slog.DebugContext(newCtx, "Instances updated", "instances", message.InstanceUpdates.UpdatedInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.UpdatedInstancesTopic, Data: message.InstanceUpdates.UpdatedInstances},
		)
	}
	if len(message.InstanceUpdates.DeletedInstances) > 0 {
		slog.DebugContext(newCtx, "Instances deleted", "instances", message.InstanceUpdates.DeletedInstances)
		w.healthWatcherService.DeleteHealthWatcher(message.InstanceUpdates.
			DeletedInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.DeletedInstancesTopic, Data: message.InstanceUpdates.DeletedInstances},
		)
	}
}
