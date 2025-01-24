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

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/uuid"
)

const (
	versionFilePath                = "/opt/app_protect/VERSION"
	releaseFilePath                = "/opt/app_protect/RELEASE"
	processName                    = "bd-socket-plugin"
	attackSignatureVersionFilePath = "/opt/app_protect/var/update_files/signatures/version"
	threatCampaignVersionFilePath  = "/opt/app_protect/var/update_files/threat_campaigns/version"
)

type (
	NginxAppProtectProcessParser struct {
		versionFilePath                string
		releaseFilePath                string
		attackSignatureVersionFilePath string
		threatCampaignVersionFilePath  string
	}
)

var _ processParser = (*NginxAppProtectProcessParser)(nil)

func NewNginxAppProtectProcessParser() *NginxAppProtectProcessParser {
	return &NginxAppProtectProcessParser{
		versionFilePath:                versionFilePath,
		releaseFilePath:                releaseFilePath,
		attackSignatureVersionFilePath: attackSignatureVersionFilePath,
		threatCampaignVersionFilePath:  threatCampaignVersionFilePath,
	}
}

func (n NginxAppProtectProcessParser) Parse(ctx context.Context, processes []*model.Process) map[string]*mpi.Instance {
	instanceMap := make(map[string]*mpi.Instance) // key is instanceID

	for _, process := range processes {
		if process.Name == processName {
			instanceID := n.instanceID(process)

			binaryPath := process.Exe
			if binaryPath == "" {
				binaryPath = strings.Split(process.Cmd, " ")[0]
			}

			instanceMap[instanceID] = &mpi.Instance{
				InstanceMeta: &mpi.InstanceMeta{
					InstanceId:   instanceID,
					InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX_APP_PROTECT,
					Version:      n.instanceVersion(ctx),
				},
				InstanceConfig: &mpi.InstanceConfig{},
				InstanceRuntime: &mpi.InstanceRuntime{
					ProcessId:  process.PID,
					BinaryPath: binaryPath,
					Details: &mpi.InstanceRuntime_NginxAppProtectRuntimeInfo{
						NginxAppProtectRuntimeInfo: &mpi.NGINXAppProtectRuntimeInfo{
							Release:                n.release(ctx),
							AttackSignatureVersion: n.attackSignatureVersion(ctx),
							ThreatCampaignVersion:  n.threatCampaignVersion(ctx),
						},
					},
					InstanceChildren: make([]*mpi.InstanceChild, 0),
				},
			}
		}
	}

	return instanceMap
}

func (n NginxAppProtectProcessParser) instanceID(process *model.Process) string {
	return uuid.Generate("%s", process.Exe)
}

func (n NginxAppProtectProcessParser) instanceVersion(ctx context.Context) string {
	version, err := os.ReadFile(n.versionFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP version file", "file_path", n.versionFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(version), "\n")
}

func (n NginxAppProtectProcessParser) release(ctx context.Context) string {
	release, err := os.ReadFile(n.releaseFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP release file", "file_path", n.releaseFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(release), "\n")
}

func (n NginxAppProtectProcessParser) attackSignatureVersion(ctx context.Context) string {
	attackSignatureVersion, err := os.ReadFile(n.attackSignatureVersionFilePath)
	if err != nil {
		slog.WarnContext(
			ctx,
			"Unable to read NAP attack signature version file",
			"file_path", n.attackSignatureVersionFilePath,
			"error", err,
		)

		return ""
	}

	return string(attackSignatureVersion)
}

func (n NginxAppProtectProcessParser) threatCampaignVersion(ctx context.Context) string {
	threatCampaignVersion, err := os.ReadFile(n.threatCampaignVersionFilePath)
	if err != nil {
		slog.WarnContext(
			ctx,
			"Unable to read NAP threat campaign version file",
			"file_path", n.threatCampaignVersionFilePath,
			"error", err,
		)

		return ""
	}

	return string(threatCampaignVersion)
}
