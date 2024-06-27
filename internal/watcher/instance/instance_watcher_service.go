// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"log/slog"
	"time"

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
		agentConfig       *config.Config
		processOperator   process.ProcessOperatorInterface
		processParsers    []processParser
		nginxConfigParser nginxConfigParser
		instanceCache     []*mpi.Instance
		nginxConfigCache  map[string]*model.NginxConfigContext // key is instanceID
		executer          exec.ExecInterface
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
		instanceCache:     []*mpi.Instance{},
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

	instanceWatcherTicker := time.NewTicker(monitoringFrequency)
	defer instanceWatcherTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(instancesChannel)
			close(nginxConfigContextChannel)

			return
		case <-instanceWatcherTicker.C:
			iw.checkForUpdates(ctx, instancesChannel, nginxConfigContextChannel)
		}
	}
}

func (iw *InstanceWatcherService) checkForUpdates(
	ctx context.Context,
	instancesChannel chan<- InstanceUpdatesMessage,
	nginxConfigContextChannel chan<- NginxConfigContextMessage,
) {
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

		if instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX ||
			instanceType == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
			nginxConfigContext, parseErr := iw.parseNginxInstanceConfig(newCtx, newInstance)
			if parseErr != nil {
				slog.WarnContext(
					ctx,
					"Parsing NGINX instance config",
					"config_path", newInstance.GetInstanceRuntime().GetConfigPath(),
					"instance_id", newInstance.GetInstanceMeta().GetInstanceId(),
					"error", parseErr,
				)
			} else {
				iw.updateNginxInstanceRuntime(newInstance, nginxConfigContext)

				nginxConfigContextChannel <- NginxConfigContextMessage{
					CorrelationID:      correlationID,
					NginxConfigContext: nginxConfigContext,
				}
			}
		}
	}

	if len(instanceUpdates.NewInstances) > 0 || len(instanceUpdates.DeletedInstances) > 0 ||
		len(instanceUpdates.UpdatedInstances) > 0 {
		instancesChannel <- InstanceUpdatesMessage{
			CorrelationID:   correlationID,
			InstanceUpdates: instanceUpdates,
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
	instancesFound := []*mpi.Instance{iw.agentInstance(ctx)}

	for _, processParser := range iw.processParsers {
		instances := processParser.Parse(ctx, processes)
		for _, instance := range instances {
			instancesFound = append(instancesFound, instance)
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

func compareInstances(oldInstances, instances []*mpi.Instance) (
	newInstances, updatedInstances, deletedInstances []*mpi.Instance,
) {
	instancesMap := make(map[string]*mpi.Instance)
	oldInstancesMap := make(map[string]*mpi.Instance)
	updatedInstancesMap := make(map[string]*mpi.Instance)
	updatedOldInstancesMap := make(map[string]*mpi.Instance)

	for _, instance := range instances {
		instancesMap[instance.GetInstanceMeta().GetInstanceId()] = instance
	}

	for _, oldInstance := range oldInstances {
		oldInstancesMap[oldInstance.GetInstanceMeta().GetInstanceId()] = oldInstance
	}

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

func (iw *InstanceWatcherService) parseNginxInstanceConfig(
	ctx context.Context,
	instance *mpi.Instance,
) (*model.NginxConfigContext, error) {
	nginxConfigContext, parseErr := iw.nginxConfigParser.Parse(ctx, instance)
	if parseErr != nil {
		return nil, parseErr
	}

	iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext

	return nginxConfigContext, nil
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
