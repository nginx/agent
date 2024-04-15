// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	// "bufio"
	"context"
	// "fmt"
	// "io"
	// "log/slog"
	// "os"
	// "strings"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host"
	// gopsutilHost "github.com/shirou/gopsutil/v3/host"
)

// const (
// 	versionID     = "VERSION_ID"
// 	version       = "VERSION"
// 	codeName      = "VERSION_CODENAME"
// 	id            = "ID"
// 	name          = "NAME"
// 	releaseFile   = "/etc/os-release"
// 	fieldPosition = 2
// )

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ResourceServiceInterface
type ResourceServiceInterface interface {
	GetResource(ctx context.Context) *v1.Resource
}

type ResourceService struct {
	info     host.InfoInterface
	resource *v1.Resource
}

func NewResourceService() *ResourceService {
	resource := &v1.Resource{
		ResourceId: "",
		// the first instance is the Agent
		Instances: []*v1.Instance{
			{
				InstanceMeta: &v1.InstanceMeta{
					// InstanceId:   gc.config.UUID,
					InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
					// Version:      gc.config.Version,
				},
				InstanceConfig: &v1.InstanceConfig{},
			},
		},
		// Info: host.NewInfo(),
	}

	return &ResourceService{
		info:     host.NewInfo(),
		resource: resource,
	}
}

func (rs *ResourceService) GetResource(ctx context.Context) *v1.Resource {
	resource := &v1.Resource{
		Instances: []*v1.Instance{},
	}

	if rs.info.IsContainer() {
		resource.Info = rs.info.ContainerInfo()
		resource.ResourceId = resource.GetContainerInfo().GetContainerId()
	} else {
		resource.Info = rs.info.HostInfo(ctx)
		resource.ResourceId = resource.GetHostInfo().GetHostId()
	}

	// resource.Info = rs.info.GetHostInfo(ctx)
	// resource.ResourceId = resource.GetHostInfo().GetHostId()
	// resource.Instances = append(resource.Instances, host)

	return resource
}

// func (rs *ResourceService) getHostReleaseInfo(ctx context.Context) (release *v1.ReleaseInfo) {
// 	hostInfo, err := gopsutilHost.InfoWithContext(ctx)
// 	if err != nil {
// 		slog.Warn("Could not read release information for host: ", "error", err)

// 		return &v1.ReleaseInfo{}
// 	}

// 	return &v1.ReleaseInfo{
// 		VersionId: hostInfo.PlatformVersion,
// 		Version:   hostInfo.KernelVersion,
// 		Codename:  hostInfo.OS,
// 		Name:      hostInfo.PlatformFamily,
// 		Id:        hostInfo.Platform,
// 	}
// }

// func (rs *ResourceService) releaseInfo(ctx context.Context, osReleaseFile string) (release *v1.ReleaseInfo) {
// 	hostReleaseInfo := rs.getHostReleaseInfo(ctx)
// 	osRelease, err := rs.getOsRelease(osReleaseFile)
// 	if err != nil {
// 		slog.Warn("ould not read from osRelease file", "error", err)
// 		return hostReleaseInfo
// 	}

// 	return rs.mergeHostAndOsReleaseInfo(hostReleaseInfo, osRelease)
// }

// func (rs *ResourceService) mergeHostAndOsReleaseInfo(hostReleaseInfo *v1.ReleaseInfo,
// 	osReleaseInfo map[string]string,
// ) (release *v1.ReleaseInfo) {
// 	// override os-release info with host info,
// 	// if os-release info is empty.
// 	if len(osReleaseInfo[versionID]) == 0 {
// 		osReleaseInfo[versionID] = hostReleaseInfo.GetVersionId()
// 	}
// 	if len(osReleaseInfo[version]) == 0 {
// 		osReleaseInfo[version] = hostReleaseInfo.GetVersion()
// 	}
// 	if len(osReleaseInfo[codeName]) == 0 {
// 		osReleaseInfo[codeName] = hostReleaseInfo.GetCodename()
// 	}
// 	if len(osReleaseInfo[name]) == 0 {
// 		osReleaseInfo[name] = hostReleaseInfo.GetName()
// 	}
// 	if len(osReleaseInfo[id]) == 0 {
// 		osReleaseInfo[id] = hostReleaseInfo.GetId()
// 	}

// 	return &v1.ReleaseInfo{
// 		VersionId: osReleaseInfo[versionID],
// 		Version:   osReleaseInfo[version],
// 		Codename:  osReleaseInfo[codeName],
// 		Name:      osReleaseInfo[name],
// 		Id:        osReleaseInfo[id],
// 	}
// }

// func (rs *ResourceService) getOsRelease(path string) (map[string]string, error) {
// 	f, err := os.Open(path)
// 	if err != nil {
// 		return nil, fmt.Errorf("release file %s unreadable: %w", path, err)
// 	}

// 	defer func() {
// 		cerr := f.Close()
// 		if err == nil {
// 			err = cerr
// 		}
// 	}()

// 	info, err := rs.parseOsReleaseFile(f)
// 	if err != nil {
// 		return nil, fmt.Errorf("release file %s unparsable: %w", path, err)
// 	}

// 	return info, nil
// }

// func (rs *ResourceService) parseOsReleaseFile(reader io.Reader) (map[string]string, error) {
// 	osReleaseInfoMap := map[string]string{"NAME": "unix"}
// 	scanner := bufio.NewScanner(reader)
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())
// 		field := strings.Split(line, "=")
// 		if len(field) < fieldPosition {
// 			continue
// 		}
// 		osReleaseInfoMap[field[0]] = strings.Trim(field[1], "\"")
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return nil, fmt.Errorf("could not parse os-release file %w", err)
// 	}

// 	return osReleaseInfoMap, nil
// }
