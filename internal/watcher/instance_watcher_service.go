// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"google.golang.org/protobuf/types/known/structpb"
)

const defaultAgentPath = "/run/nginx-agent"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . processParser

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . nginxConfigParser

type (
	processOperator interface {
		Processes(ctx context.Context) ([]*model.Process, error)
	}

	processParser interface {
		Parse(ctx context.Context, processes []*model.Process) []*v1.Instance
	}

	nginxConfigParser interface {
		Parse(ctx context.Context, instance *v1.Instance) (*model.NginxConfigContext, error)
	}

	InstanceWatcherService struct {
		agentConfig       *config.Config
		processOperator   processOperator
		processParsers    []processParser
		nginxConfigParser nginxConfigParser
		instanceCache     []*v1.Instance
		nginxConfigCache  map[string]*model.NginxConfigContext // key is instanceID
		executer          exec.ExecInterface
	}

	InstanceUpdates struct {
		newInstances     []*v1.Instance
		deletedInstances []*v1.Instance
	}

	InstanceUpdatesMessage struct {
		correlationID   slog.Attr
		instanceUpdates InstanceUpdates
	}

	NginxConfigContextMessage struct {
		correlationID      slog.Attr
		nginxConfigContext *model.NginxConfigContext
	}
)

func NewInstanceWatcherService(agentConfig *config.Config) *InstanceWatcherService {
	return &InstanceWatcherService{
		agentConfig:     agentConfig,
		processOperator: NewProcessOperator(),
		processParsers: []processParser{
			NewNginxProcessParser(),
		},
		nginxConfigParser: NewNginxConfigParser(agentConfig),
		instanceCache:     []*v1.Instance{},
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
	correlationID := logger.GenerateCorrelationID()
	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, correlationID)

	instanceUpdates, err := iw.instanceUpdates(newCtx)
	if err != nil {
		slog.ErrorContext(newCtx, "Instance watcher updates", "error", err)
	}

	for _, newInstance := range instanceUpdates.newInstances {
		instanceType := newInstance.GetInstanceMeta().GetInstanceType()

		if instanceType == v1.InstanceMeta_INSTANCE_TYPE_NGINX ||
			instanceType == v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
			nginxConfigContext := iw.parseNginxInstanceConfig(newCtx, newInstance)
			iw.updateNginxInstanceRuntime(newInstance, nginxConfigContext)

			nginxConfigContextChannel <- NginxConfigContextMessage{
				correlationID:      correlationID,
				nginxConfigContext: nginxConfigContext,
			}
		}
	}

	if len(instanceUpdates.newInstances) > 0 || len(instanceUpdates.deletedInstances) > 0 {
		instancesChannel <- InstanceUpdatesMessage{
			correlationID:   correlationID,
			instanceUpdates: instanceUpdates,
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
	instancesFound := []*v1.Instance{iw.agentInstance(ctx)}

	for _, processParser := range iw.processParsers {
		instancesFound = append(instancesFound, processParser.Parse(ctx, processes)...)
	}

	newInstances, deletedInstances := compareInstances(iw.instanceCache, instancesFound)

	instanceUpdates.newInstances = newInstances
	instanceUpdates.deletedInstances = deletedInstances

	iw.instanceCache = instancesFound

	return instanceUpdates, nil
}

func (iw *InstanceWatcherService) agentInstance(ctx context.Context) *v1.Instance {
	processPath, err := iw.executer.Executable()
	if err != nil {
		processPath = defaultAgentPath
		slog.WarnContext(ctx, "Unable to read process location, defaulting to /var/run/nginx-agent", "error", err)
	}

	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   iw.agentConfig.UUID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      iw.agentConfig.Version,
		},
		InstanceConfig: &v1.InstanceConfig{
			Actions: []*v1.InstanceAction{},
			Config: &v1.InstanceConfig_AgentConfig{
				AgentConfig: &v1.AgentConfig{
					Command:           &v1.CommandServer{},
					Metrics:           &v1.MetricsServer{},
					File:              &v1.FileServer{},
					Labels:            []*structpb.Struct{},
					Features:          []string{},
					MessageBufferSize: "",
				},
			},
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  iw.executer.ProcessID(),
			BinaryPath: processPath,
			ConfigPath: iw.agentConfig.Path,
			Details:    nil,
		},
	}
}

func compareInstances(oldInstances, instances []*v1.Instance) (newInstances, deletedInstances []*v1.Instance) {
	instancesMap := make(map[int32]*v1.Instance)
	oldInstancesMap := make(map[int32]*v1.Instance)

	for _, instance := range instances {
		instancesMap[instance.GetInstanceRuntime().GetProcessId()] = instance
	}

	for _, oldInstance := range oldInstances {
		oldInstancesMap[oldInstance.GetInstanceRuntime().GetProcessId()] = oldInstance
	}

	for pid, instance := range instancesMap {
		_, ok := oldInstancesMap[pid]
		if !ok {
			newInstances = append(newInstances, instance)
		}
	}

	for pid, oldInstance := range oldInstancesMap {
		_, ok := instancesMap[pid]
		if !ok {
			deletedInstances = append(deletedInstances, oldInstance)
		}
	}

	return newInstances, deletedInstances
}

func (iw *InstanceWatcherService) updateNginxInstanceRuntime(
	instance *v1.Instance,
	nginxConfigContext *model.NginxConfigContext,
) {
	instanceType := instance.GetInstanceMeta().GetInstanceType()

	if instanceType == v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
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
	instance *v1.Instance,
) *model.NginxConfigContext {
	nginxConfigContext, parseErr := iw.nginxConfigParser.Parse(ctx, instance)
	if parseErr != nil {
		slog.WarnContext(
			ctx,
			"Parsing NGINX instance config",
			"config_path", instance.GetInstanceRuntime().GetConfigPath(),
			"instance_id", instance.GetInstanceMeta().GetInstanceId(),
			"error", parseErr,
		)
	}

	iw.nginxConfigCache[nginxConfigContext.InstanceID] = nginxConfigContext

	return nginxConfigContext
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
