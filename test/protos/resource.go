// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import "github.com/nginx/agent/v3/api/grpc/mpi/v1"

func ContainerizedResource() *v1.Resource {
	return &v1.Resource{
		ResourceId: ContainerInfo().GetContainerId(),
		Instances: []*v1.Instance{
			NginxOssInstance([]string{}),
		},
		Info: &v1.Resource_ContainerInfo{
			ContainerInfo: ContainerInfo(),
		},
	}
}

func HostResource() *v1.Resource {
	return &v1.Resource{
		ResourceId: HostInfo().GetHostId(),
		Instances: []*v1.Instance{
			NginxOssInstance([]string{}),
		},
		Info: &v1.Resource_HostInfo{
			HostInfo: HostInfo(),
		},
	}
}

func HostInfo() *v1.HostInfo {
	return &v1.HostInfo{
		HostId:      "1234",
		Hostname:    "test-host",
		ReleaseInfo: ReleaseInfo(),
	}
}

func ReleaseInfo() *v1.ReleaseInfo {
	return &v1.ReleaseInfo{
		Codename:  "Focal Fossa",
		Id:        "ubuntu",
		Name:      "Ubuntu 20.04.3 LTS",
		VersionId: "20.04.3",
		Version:   "",
	}
}

func ContainerInfo() *v1.ContainerInfo {
	return &v1.ContainerInfo{
		ContainerId: "f43f5eg54g54g54",
	}
}

func InstanceHealths() []*v1.InstanceHealth {
	return []*v1.InstanceHealth{
		{
			InstanceId:           NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			InstanceHealthStatus: v1.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY,
			Description:          "healthy",
		},
		{
			InstanceId:           NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			InstanceHealthStatus: v1.InstanceHealth_INSTANCE_HEALTH_STATUS_UNHEALTHY,
			Description:          "unhealthy",
		},
	}
}
