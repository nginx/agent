// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/nginx/agent/v3/pkg/nginxprocess"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/pkg/id"
)

const (
	versionFilePath                = "/opt/app_protect/VERSION"
	releaseFilePath                = "/opt/app_protect/RELEASE"
	processName                    = "bd-socket-plugin"
	attackSignatureVersionFilePath = "/opt/app_protect/var/update_files/signatures/version"
	threatCampaignVersionFilePath  = "/opt/app_protect/var/update_files/threat_campaigns/version"
)

type (
	NginxAppProtectParser struct {
		versionFilePath                string
		releaseFilePath                string
		attackSignatureVersionFilePath string
		threatCampaignVersionFilePath  string
	}
)

func NewNginxAppProtectParser() *NginxAppProtectParser {
	return &NginxAppProtectParser{
		versionFilePath:                versionFilePath,
		releaseFilePath:                releaseFilePath,
		attackSignatureVersionFilePath: attackSignatureVersionFilePath,
		threatCampaignVersionFilePath:  threatCampaignVersionFilePath,
	}
}

func (n NginxAppProtectParser) Parse(
	ctx context.Context,
) map[string]*mpi.Instance {
	instanceMap := make(map[string]*mpi.Instance) // key is instanceID

	if n.isNAPInstance() {
		instanceID := id.Generate("")

		instanceMap[instanceID] = &mpi.Instance{
			InstanceMeta: &mpi.InstanceMeta{
				InstanceId:   instanceID,
				InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX_APP_PROTECT,
				Version:      n.instanceVersion(ctx),
			},
			InstanceConfig: &mpi.InstanceConfig{},
			InstanceRuntime: &mpi.InstanceRuntime{
				ProcessId:  0,
				BinaryPath: "",
				ConfigPath: "",
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

	return instanceMap
}

func (n NginxAppProtectParser) isNAPInstance() bool {
	_, errVersion := os.Stat(n.versionFilePath)
	_, errRelease := os.Stat(n.releaseFilePath)
	if errors.Is(errVersion, os.ErrNotExist) || errors.Is(errRelease, os.ErrNotExist) {
		return false
	}
	return true
}

func (n NginxAppProtectParser) instanceID(process *nginxprocess.Process) string {
	return id.Generate("%s", process.Exe)
}

func (n NginxAppProtectParser) instanceVersion(ctx context.Context) string {
	version, err := os.ReadFile(n.versionFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP version file", "file_path", n.versionFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(version), "\n")
}

func (n NginxAppProtectParser) release(ctx context.Context) string {
	release, err := os.ReadFile(n.releaseFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP release file", "file_path", n.releaseFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(release), "\n")
}

func (n NginxAppProtectParser) attackSignatureVersion(ctx context.Context) string {
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

func (n NginxAppProtectParser) threatCampaignVersion(ctx context.Context) string {
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
