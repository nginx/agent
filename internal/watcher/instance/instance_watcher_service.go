// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nginx/agent/v3/pkg/host/exec"

	"github.com/nginx/agent/v3/internal/datasource/proto"

	parser "github.com/nginx/agent/v3/internal/datasource/config"

	"github.com/nginx/agent/v3/pkg/nginxprocess"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/watcher/process"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
)

const defaultAgentPath = "/run/nginx-agent"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processParser

type (
	processParser interface {
		Parse(ctx context.Context, processes []*nginxprocess.Process) map[string]*mpi.Instance
	}

	InstanceWatcherService struct {
		processOperator                process.ProcessOperatorInterface
		nginxAppProtectInstanceWatcher *NginxAppProtectInstanceWatcher
		nginxConfigParser              parser.ConfigParser
		executer                       exec.ExecInterface
		enabled                        *atomic.Bool
		agentConfig                    *config.Config
		instanceCache                  map[string]*mpi.Instance
		nginxConfigCache               map[string]*model.NginxConfigContext
		instancesChannel               chan<- InstanceUpdatesMessage
		nginxConfigContextChannel      chan<- NginxConfigContextMessage
		nginxParser                    processParser
		cacheMutex                     sync.Mutex
	}

	InstanceUpdates struct {
		UpdatedInstances []*mpi.Instance
	}

	InstanceUpdatesMessage struct {
		CorrelationID   slog.Attr
		InstanceUpdates InstanceUpdates
	}

	NginxConfigContextMessage struct {
		CorrelationID      slog.Attr
		NginxConfigContext *model.NginxConfigContext
	}
)

func NewInstanceWatcherService(agentConfig *config.Config, watcher *NginxAppProtectInstanceWatcher) *InstanceWatcherService {
	enabled := &atomic.Bool{}
	enabled.Store(true)

	return &InstanceWatcherService{
		agentConfig:                    agentConfig,
		nginxAppProtectInstanceWatcher: watcher,
		processOperator:                process.NewProcessOperator(),
		nginxParser:                    NewNginxProcessParser(),
		nginxConfigParser:              parser.NewNginxConfigParser(agentConfig),
		instanceCache:                  make(map[string]*mpi.Instance),
		cacheMutex:                     sync.Mutex{},
		nginxConfigCache:               make(map[string]*model.NginxConfigContext),
		executer:                       &exec.Exec{},
		enabled:                        enabled,
	}
}

func (iw *InstanceWatcherService) SetEnabled(enabled bool) {
	iw.enabled.Store(enabled)
}

func (iw *InstanceWatcherService) Watch(
	ctx context.Context,
	instancesChannel chan<- InstanceUpdatesMessage,
	nginxConfigContextChannel chan<- NginxConfigContextMessage,
) {
	monitoringFrequency := iw.agentConfig.Watchers.InstanceWatcher.MonitoringFrequency
	slog.DebugContext(ctx, "Starting instance watcher monitoring", "monitoring_frequency", monitoringFrequency)

	iw.instancesChannel = instancesChannel
	iw.nginxConfigContextChannel = nginxConfigContextChannel

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(instancesChannel)
			close(nginxConfigContextChannel)

			return
		case <-instanceWatcherTicker.C:
			if iw.enabled.Load() {
				iw.checkForUpdates(ctx)
			} else {
				slog.DebugContext(ctx, "Skipping check for instance updates, instance watcher is disabled")
			}
		}
	}
}

func (iw *InstanceWatcherService) ReparseConfigs(ctx context.Context) {
	slog.DebugContext(ctx, "Reparsing all instance configurations")
	for _, instance := range iw.instanceCache {
		nginxConfigContext := &model.NginxConfigContext{}
		var parseErr error
		slog.DebugContext(
			ctx,
			"Reparsing NGINX instance config",
			"instance_id", instance.GetInstanceMeta().GetInstanceId(),
		)

		if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX ||
			instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
			nginxConfigContext, parseErr = iw.nginxConfigParser.Parse(ctx, instance)
			if parseErr != nil {
				slog.WarnContext(
					ctx,
					"Failed to parse NGINX instance config",
					"config_path", instance.GetInstanceRuntime().GetConfigPath(),
					"instance_id", instance.GetInstanceMeta().GetInstanceId(),
					"error", parseErr,
				)

				return
			}
		}

		iw.HandleNginxConfigContextUpdate(ctx, instance.GetInstanceMeta().GetInstanceId(), nginxConfigContext)
	}
	slog.DebugContext(ctx, "Finished reparsing all instance configurations")
}

func (iw *InstanceWatcherService) HandleNginxConfigContextUpdate(ctx context.Context, instanceID string,
	nginxConfigContext *model.NginxConfigContext,
) {
	iw.cacheMutex.Lock()
	defer iw.cacheMutex.Unlock()

	updatesRequired := false
	instance := iw.instanceCache[instanceID]
	instanceType := instance.GetInstanceMeta().GetInstanceType()
	correlationID := logger.CorrelationIDAttr(ctx)

	if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX ||
		instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		iw.sendNginxConfigContextUpdate(ctx, nginxConfigContext)
		iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext
		updatesRequired = proto.UpdateNginxInstanceRuntime(instance, nginxConfigContext)
	}

	if updatesRequired {
		instanceUpdates := InstanceUpdates{}
		instanceUpdates.UpdatedInstances = append(instanceUpdates.UpdatedInstances, instance)
		iw.instancesChannel <- InstanceUpdatesMessage{
			CorrelationID:   correlationID,
			InstanceUpdates: instanceUpdates,
		}
	}
}

func (iw *InstanceWatcherService) checkForUpdates(
	ctx context.Context,
) {
	var instancesToParse []*mpi.Instance
	correlationID := logger.GenerateCorrelationID()
	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, correlationID)

	instanceUpdates, err := iw.instanceUpdates(newCtx)
	if err != nil {
		slog.ErrorContext(newCtx, "Instance watcher updates", "error", err)
	}

	instancesToParse = append(instancesToParse, instanceUpdates.UpdatedInstances...)

	for _, newInstance := range instancesToParse {
		instanceType := newInstance.GetInstanceMeta().GetInstanceType()
		slog.DebugContext(
			newCtx,
			"Parsing instance config",
			"instance_id", newInstance.GetInstanceMeta().GetInstanceId(),
			"instance_type", instanceType,
		)

		if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX ||
			instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
			nginxConfigContext, parseErr := iw.nginxConfigParser.Parse(newCtx, newInstance)
			if parseErr != nil {
				slog.WarnContext(
					newCtx,
					"Unable to parse NGINX instance config",
					"config_path", newInstance.GetInstanceRuntime().GetConfigPath(),
					"instance_id", newInstance.GetInstanceMeta().GetInstanceId(),
					"instance_type", instanceType,
					"error", parseErr,
				)
			} else {
				iw.cacheMutex.Lock()
				iw.sendNginxConfigContextUpdate(newCtx, nginxConfigContext)
				proto.UpdateNginxInstanceRuntime(newInstance, nginxConfigContext)

				iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext
				iw.instanceCache[newInstance.GetInstanceMeta().GetInstanceId()] = newInstance
				iw.cacheMutex.Unlock()
			}
		}
	}

	if iw.nginxAppProtectInstanceWatcher.NginxAppProtectInstance() != nil {
		slog.DebugContext(ctx, "Adding nginx app protect instance to instance list")
		instanceUpdates.UpdatedInstances = append(instanceUpdates.UpdatedInstances, iw.nginxAppProtectInstanceWatcher.NginxAppProtectInstance())
	}

	if len(instanceUpdates.UpdatedInstances) > 0 {
		iw.instancesChannel <- InstanceUpdatesMessage{
			CorrelationID:   correlationID,
			InstanceUpdates: instanceUpdates,
		}
	}
}

func (iw *InstanceWatcherService) sendNginxConfigContextUpdate(
	ctx context.Context,
	nginxConfigContext *model.NginxConfigContext,
) {
	if iw.nginxConfigCache[nginxConfigContext.InstanceID] == nil ||
		!iw.nginxConfigCache[nginxConfigContext.InstanceID].Equal(nginxConfigContext) {
		slog.DebugContext(
			ctx,
			"New NGINX config context",
			"instance_id", nginxConfigContext.InstanceID,
			"nginx_config_context", nginxConfigContext,
		)

		iw.nginxConfigContextChannel <- NginxConfigContextMessage{
			CorrelationID:      logger.CorrelationIDAttr(ctx),
			NginxConfigContext: nginxConfigContext,
		}
	}
}

func (iw *InstanceWatcherService) instanceUpdates(ctx context.Context) (
	instanceUpdates InstanceUpdates,
	err error,
) {
	iw.cacheMutex.Lock()
	defer iw.cacheMutex.Unlock()
	nginxProcesses, err := iw.processOperator.Processes(ctx)
	if err != nil {
		return instanceUpdates, err
	}

	// NGINX Agent is always the first instance in the list
	instancesFound := make(map[string]*mpi.Instance)
	agentInstance := iw.agentInstance(ctx)
	instancesFound[agentInstance.GetInstanceMeta().GetInstanceId()] = agentInstance

	nginxInstances := iw.nginxParser.Parse(ctx, nginxProcesses)
	for _, instance := range nginxInstances {
		instancesFound[instance.GetInstanceMeta().GetInstanceId()] = instance
	}

	if areInstanceDifferent(iw.instanceCache, instancesFound) {
		var updatedInstances []*mpi.Instance
		for _, instance := range instancesFound {
			updatedInstances = append(updatedInstances, instance)
		}

		instanceUpdates.UpdatedInstances = updatedInstances
	}

	iw.instanceCache = instancesFound

	return instanceUpdates, nil
}

func (iw *InstanceWatcherService) agentInstance(ctx context.Context) *mpi.Instance {
	processPath, err := iw.executer.Executable()
	if err != nil {
		processPath = defaultAgentPath
		slog.WarnContext(ctx, "Unable to read process location, defaulting to /var/run/nginx-agent", "error", err)
	}

	labels, convertErr := mpi.ConvertToStructs(iw.agentConfig.Labels)
	if convertErr != nil {
		slog.WarnContext(ctx, "Unable to convert config to labels structure", "error", convertErr)
	}

	instance := &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   iw.agentConfig.UUID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      iw.agentConfig.Version,
		},
		InstanceConfig: &mpi.InstanceConfig{
			Actions: []*mpi.InstanceAction{},
			Config: &mpi.InstanceConfig_AgentConfig{
				AgentConfig: &mpi.AgentConfig{
					Command:           config.ToCommandProto(iw.agentConfig.Command),
					Metrics:           &mpi.MetricsServer{},
					File:              &mpi.FileServer{},
					Labels:            labels,
					Features:          iw.agentConfig.Features,
					MessageBufferSize: "",
					Log:               config.ToAgentConfigLogProto(iw.agentConfig.Log),
				},
			},
		},
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  iw.executer.ProcessID(),
			BinaryPath: processPath,
			ConfigPath: iw.agentConfig.Path,
			Details:    nil,
		},
	}

	if iw.agentConfig.IsAuxiliaryCommandGrpcClientConfigured() {
		instance.GetInstanceConfig().GetAgentConfig().AuxiliaryCommand = config.
			ToAuxiliaryCommandServerProto(iw.agentConfig.AuxiliaryCommand)
	}

	return instance
}

func areInstanceDifferent(oldInstancesMap, instancesMap map[string]*mpi.Instance) bool {
	updatedInstancesMap := make(map[string]*mpi.Instance)
	updatedOldInstancesMap := make(map[string]*mpi.Instance)

	for instanceID, instance := range instancesMap {
		_, ok := oldInstancesMap[instanceID]
		if !ok {
			return true
		} else {
			updatedInstancesMap[instanceID] = instance
		}
	}

	for instanceID, oldInstance := range oldInstancesMap {
		_, ok := instancesMap[instanceID]
		if !ok {
			return true
		} else {
			updatedOldInstancesMap[instanceID] = oldInstance
		}
	}

	return checkForProcessChanges(updatedInstancesMap, updatedOldInstancesMap)
}

func checkForProcessChanges(
	updatedInstancesMap map[string]*mpi.Instance,
	updatedOldInstancesMap map[string]*mpi.Instance,
) (updated bool) {
	updated = false
	for instanceID, instance := range updatedInstancesMap {
		oldInstance := updatedOldInstancesMap[instanceID]
		if !areInstancesEqual(oldInstance.GetInstanceRuntime(), instance.GetInstanceRuntime()) {
			return true
		}
	}

	return updated
}

func areInstancesEqual(oldRuntime, currentRuntime *mpi.InstanceRuntime) (equal bool) {
	if oldRuntime.GetProcessId() != currentRuntime.GetProcessId() {
		return false
	}

	oldRuntimeChildren := oldRuntime.GetInstanceChildren()
	currentRuntimeChildren := currentRuntime.GetInstanceChildren()

	if len(oldRuntimeChildren) != len(currentRuntimeChildren) {
		return false
	}

	for _, oldChild := range oldRuntimeChildren {
		childFound := false
		for _, currentChild := range currentRuntimeChildren {
			if oldChild.GetProcessId() == currentChild.GetProcessId() {
				childFound = true
				break
			}
		}

		if !childFound {
			return false
		}
	}

	return true
}
