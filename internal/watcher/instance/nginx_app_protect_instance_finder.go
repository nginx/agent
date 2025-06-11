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

	"github.com/nginx/agent/v3/pkg/id"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

const (
	versionFilePath                = "/opt/app_protect/VERSION"
	releaseFilePath                = "/opt/app_protect/RELEASE"
	attackSignatureVersionFilePath = "/opt/app_protect/var/update_files/signatures/version"
	threatCampaignVersionFilePath  = "/opt/app_protect/var/update_files/threat_campaigns/version"
	enforcerEngineVersionFilePath  = "/opt/app_protect/bd_config/enforcer.version"
)

type (
	NginxAppProtectInstanceFinder struct {
		versionFilePath                string
		releaseFilePath                string
		attackSignatureVersionFilePath string
		threatCampaignVersionFilePath  string
		enforcerEngineVersionFilePath  string
	}
)

var _ instanceFinder = (*NginxAppProtectInstanceFinder)(nil)

func NewNginxAppProtectInstanceFinder() *NginxAppProtectInstanceFinder {
	return &NginxAppProtectInstanceFinder{
		versionFilePath:                versionFilePath,
		releaseFilePath:                releaseFilePath,
		attackSignatureVersionFilePath: attackSignatureVersionFilePath,
		threatCampaignVersionFilePath:  threatCampaignVersionFilePath,
		enforcerEngineVersionFilePath:  enforcerEngineVersionFilePath,
	}
}

func (n NginxAppProtectInstanceFinder) Find(ctx context.Context) *mpi.Instance {
	if !n.isNginxAppProtectPresent() {
		return nil
	}

	return &mpi.Instance{
		InstanceMeta: &mpi.InstanceMeta{
			InstanceId:   id.Generate("%s", n.versionFilePath),
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
					EnforcerEngineVersion:  n.enforcerEngineVersion(ctx),
				},
			},
			InstanceChildren: make([]*mpi.InstanceChild, 0),
		},
	}
}

func (n NginxAppProtectInstanceFinder) isNginxAppProtectPresent() bool {
	_, errVersion := os.Stat(n.versionFilePath)
	_, errRelease := os.Stat(n.releaseFilePath)

	if errors.Is(errVersion, os.ErrNotExist) || errors.Is(errRelease, os.ErrNotExist) {
		return false
	}

	return true
}

func (n NginxAppProtectInstanceFinder) instanceVersion(ctx context.Context) string {
	version, err := os.ReadFile(n.versionFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP version file", "file_path", n.versionFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(version), "\n")
}

func (n NginxAppProtectInstanceFinder) release(ctx context.Context) string {
	release, err := os.ReadFile(n.releaseFilePath)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NAP release file", "file_path", n.releaseFilePath, "error", err)
		return ""
	}

	return strings.TrimSuffix(string(release), "\n")
}

func (n NginxAppProtectInstanceFinder) attackSignatureVersion(ctx context.Context) string {
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

func (n NginxAppProtectInstanceFinder) threatCampaignVersion(ctx context.Context) string {
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

func (n NginxAppProtectInstanceFinder) enforcerEngineVersion(ctx context.Context) string {
	enforcerEngineVersion, err := os.ReadFile(n.enforcerEngineVersionFilePath)
	if err != nil {
		slog.WarnContext(
			ctx,
			"Unable to read NAP enforcer engine version file",
			"file_path", n.enforcerEngineVersionFilePath,
			"error", err,
		)

		return ""
	}

	return string(enforcerEngineVersion)
}
