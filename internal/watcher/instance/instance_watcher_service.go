// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/watcher/process"
	"google.golang.org/protobuf/types/known/structpb"
)

const defaultAgentPath = "/run/nginx-agent"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processParser

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . nginxConfigParser

type (
	processParser interface {
		Parse(ctx context.Context, processes []*model.Process) map[string]*mpi.Instance
	}

	nginxConfigParser interface {
		Parse(ctx context.Context, instance *mpi.Instance) (*model.NginxConfigContext, error)
	}

	InstanceWatcherService struct {
		processOperator           process.ProcessOperatorInterface
		nginxConfigParser         nginxConfigParser
		executer                  exec.ExecInterface
		agentConfig               *config.Config
		instanceCache             map[string]*mpi.Instance
		nginxConfigCache          map[string]*model.NginxConfigContext
		instancesChannel          chan<- InstanceUpdatesMessage
		nginxConfigContextChannel chan<- NginxConfigContextMessage
		processParsers            []processParser
		cacheMutex                sync.Mutex
	}

	InstanceUpdates struct {
		NewInstances     []*mpi.Instance
		UpdatedInstances []*mpi.Instance
		DeletedInstances []*mpi.Instance
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

func NewInstanceWatcherService(agentConfig *config.Config) *InstanceWatcherService {
	return &InstanceWatcherService{
		agentConfig:     agentConfig,
		processOperator: process.NewProcessOperator(),
		processParsers: []processParser{
			NewNginxProcessParser(),
		},
		nginxConfigParser: NewNginxConfigParser(agentConfig),
		instanceCache:     make(map[string]*mpi.Instance),
		cacheMutex:        sync.Mutex{},
		nginxConfigCache:  make(map[string]*model.NginxConfigContext),
		executer:          &exec.Exec{},
	}
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
			iw.checkForUpdates(ctx)
		}
	}
}

func (iw *InstanceWatcherService) ReparseConfigs(ctx context.Context) {
	slog.DebugContext(ctx, "Reparsing all instance configurations")
	for _, instance := range iw.instanceCache {
		iw.ReparseConfig(ctx, instance)
	}
	slog.DebugContext(ctx, "Finished reparsing all instance configurations")
}

func (iw *InstanceWatcherService) ReparseConfig(ctx context.Context, instance *mpi.Instance) {
	iw.cacheMutex.Lock()
	defer iw.cacheMutex.Unlock()
	instanceType := instance.GetInstanceMeta().GetInstanceType()
	correlationID := logger.GetCorrelationIDAttr(ctx)

	if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX ||
		instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		slog.DebugContext(
			ctx,
			"Reparsing NGINX instance config",
			"instance_id", instance.GetInstanceMeta().GetInstanceId(),
		)

		nginxConfigContext, parseErr := iw.nginxConfigParser.Parse(ctx, instance)
		if parseErr != nil {
			slog.WarnContext(
				ctx,
				"Unable to parse NGINX instance config",
				"config_path", instance.GetInstanceRuntime().GetConfigPath(),
				"instance_id", instance.GetInstanceMeta().GetInstanceId(),
				"error", parseErr,
			)

			return
		}

		iw.sendNginxConfigContextUpdate(ctx, nginxConfigContext)
		iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext
		iw.updateNginxInstanceRuntime(instance, nginxConfigContext)
	}

	if !proto.Equal(instance, iw.instanceCache[instance.GetInstanceMeta().GetInstanceId()]) {
		iw.instanceCache[instance.GetInstanceMeta().GetInstanceId()] = instance
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
	iw.cacheMutex.Lock()
	defer iw.cacheMutex.Unlock()
	var instancesToParse []*mpi.Instance
	correlationID := logger.GenerateCorrelationID()
	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, correlationID)

	instanceUpdates, err := iw.instanceUpdates(newCtx)
	if err != nil {
		slog.ErrorContext(newCtx, "Instance watcher updates", "error", err)
	}

	instancesToParse = append(instancesToParse, instanceUpdates.UpdatedInstances...)
	instancesToParse = append(instancesToParse, instanceUpdates.NewInstances...)

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
				iw.sendNginxConfigContextUpdate(newCtx, nginxConfigContext)
				iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext
				iw.updateNginxInstanceRuntime(newInstance, nginxConfigContext)
				iw.instanceCache[newInstance.GetInstanceMeta().GetInstanceId()] = newInstance
			}
		}
	}

	if len(instanceUpdates.NewInstances) > 0 || len(instanceUpdates.DeletedInstances) > 0 ||
		len(instanceUpdates.UpdatedInstances) > 0 {
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
			CorrelationID:      logger.GetCorrelationIDAttr(ctx),
			NginxConfigContext: nginxConfigContext,
		}
	}
}

func (iw *InstanceWatcherService) instanceUpdates(ctx context.Context) (
	instanceUpdates InstanceUpdates,
	err error,
) {
	processes, err := iw.processOperator.Processes(ctx)
	if err != nil {
		return instanceUpdates, err
	}

	// NGINX Agent is always the first instance in the list
	instancesFound := make(map[string]*mpi.Instance)
	agentInstance := iw.agentInstance(ctx)
	instancesFound[agentInstance.GetInstanceMeta().GetInstanceId()] = agentInstance

	for _, parser := range iw.processParsers {
		instances := parser.Parse(ctx, processes)
		for _, instance := range instances {
			instancesFound[instance.GetInstanceMeta().GetInstanceId()] = instance
		}
	}
	newInstances, updatedInstances, deletedInstances := compareInstances(iw.instanceCache, instancesFound)

	instanceUpdates.NewInstances = newInstances
	instanceUpdates.UpdatedInstances = updatedInstances
	instanceUpdates.DeletedInstances = deletedInstances

	iw.instanceCache = instancesFound

	return instanceUpdates, nil
}

func (iw *InstanceWatcherService) agentInstance(ctx context.Context) *mpi.Instance {
	processPath, err := iw.executer.Executable()
	if err != nil {
		processPath = defaultAgentPath
		slog.WarnContext(ctx, "Unable to read process location, defaulting to /var/run/nginx-agent", "error", err)
	}

	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   iw.agentConfig.UUID,
			InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      iw.agentConfig.Version,
		},
		InstanceConfig: &mpi.InstanceConfig{
			Actions: []*mpi.InstanceAction{},
			Config: &mpi.InstanceConfig_AgentConfig{
				AgentConfig: &mpi.AgentConfig{
					Command:           &mpi.CommandServer{},
					Metrics:           &mpi.MetricsServer{},
					File:              &mpi.FileServer{},
					Labels:            []*structpb.Struct{},
					Features:          []string{},
					MessageBufferSize: "",
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
}

func compareInstances(oldInstancesMap, instancesMap map[string]*mpi.Instance) (
	newInstances, updatedInstances, deletedInstances []*mpi.Instance,
) {
	updatedInstancesMap := make(map[string]*mpi.Instance)
	updatedOldInstancesMap := make(map[string]*mpi.Instance)

	for instanceID, instance := range instancesMap {
		_, ok := oldInstancesMap[instanceID]
		if !ok {
			newInstances = append(newInstances, instance)
		} else {
			updatedInstancesMap[instanceID] = instance
		}
	}

	for instanceID, oldInstance := range oldInstancesMap {
		_, ok := instancesMap[instanceID]
		if !ok {
			deletedInstances = append(deletedInstances, oldInstance)
		} else {
			updatedOldInstancesMap[instanceID] = oldInstance
		}
	}

	updatedInstances = checkForProcessChanges(updatedInstancesMap, updatedOldInstancesMap)

	return newInstances, updatedInstances, deletedInstances
}

func checkForProcessChanges(
	updatedInstancesMap map[string]*mpi.Instance,
	updatedOldInstancesMap map[string]*mpi.Instance,
) (updatedInstances []*mpi.Instance) {
	for instanceID, instance := range updatedInstancesMap {
		oldInstance := updatedOldInstancesMap[instanceID]
		if !areInstancesEqual(oldInstance.GetInstanceRuntime(), instance.GetInstanceRuntime()) {
			updatedInstances = append(updatedInstances, instance)
		}
	}

	return updatedInstances
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

func (iw *InstanceWatcherService) updateNginxInstanceRuntime(
	instance *mpi.Instance,
	nginxConfigContext *model.NginxConfigContext,
) {
	instanceType := instance.GetInstanceMeta().GetInstanceType()

	if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		nginxPlusRuntimeInfo := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo()

		nginxPlusRuntimeInfo.AccessLogs = convertAccessLogs(nginxConfigContext.AccessLogs)
		nginxPlusRuntimeInfo.ErrorLogs = convertErrorLogs(nginxConfigContext.ErrorLogs)
		nginxPlusRuntimeInfo.StubStatus = nginxConfigContext.StubStatus
		nginxPlusRuntimeInfo.PlusApi = nginxConfigContext.PlusAPI
	} else {
		nginxRuntimeInfo := instance.GetInstanceRuntime().GetNginxRuntimeInfo()

		nginxRuntimeInfo.AccessLogs = convertAccessLogs(nginxConfigContext.AccessLogs)
		nginxRuntimeInfo.ErrorLogs = convertErrorLogs(nginxConfigContext.ErrorLogs)
		nginxRuntimeInfo.StubStatus = nginxConfigContext.StubStatus
	}
}

func convertAccessLogs(accessLogs []*model.AccessLog) (logs []string) {
	for _, log := range accessLogs {
		logs = append(logs, log.Name)
	}

	return logs
}

func convertErrorLogs(errorLogs []*model.ErrorLog) (logs []string) {
	for _, log := range errorLogs {
		logs = append(logs, log.Name)
	}

	return logs
}
