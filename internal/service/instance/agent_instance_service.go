// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"log/slog"
	"os"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"google.golang.org/protobuf/types/known/structpb"
)

const defaultAgentPath = "/run/nginx-agent"

type NginxAgent struct {
	agentConfig *config.Config
}

func NewNginxAgent(agentConfig *config.Config) *NginxAgent {
	return &NginxAgent{
		agentConfig: agentConfig,
	}
}

func (na *NginxAgent) GetInstances(ctx context.Context, _ []*model.Process) []*v1.Instance {
	processPath, err := os.Executable()
	if err != nil {
		processPath = defaultAgentPath
		slog.WarnContext(ctx, "Unable to read process location, defaulting to /var/run/nginx-agent", "error", err)
	}

	instance := &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   na.agentConfig.UUID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
			Version:      na.agentConfig.Version,
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
			ProcessId:  int32(os.Getpid()),
			BinaryPath: processPath,
			ConfigPath: na.agentConfig.Path,
			Details:    nil,
		},
	}

	return []*v1.Instance{instance}
}
