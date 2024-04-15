// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import "github.com/nginx/agent/v3/api/grpc/mpi/v1"

func GetContainerizedResource() *v1.Resource {
	return &v1.Resource{
		ResourceId: "f43f5eg54g54g54",
		Instances:  []*v1.Instance{},
		Info: &v1.Resource_ContainerInfo{
			ContainerInfo: &v1.ContainerInfo{
				ContainerId: "f43f5eg54g54g54",
			},
		},
	}
}

func GetHostResource() *v1.Resource {
	return &v1.Resource{
		ResourceId: "1234",
		Instances:  []*v1.Instance{},
		Info: &v1.Resource_HostInfo{
			HostInfo: &v1.HostInfo{
				HostId:   "1234",
				Hostname: "test-host",
				ReleaseInfo: &v1.ReleaseInfo{
					Codename:  "Focal Fossa",
					Id:        "ubuntu",
					Name:      "Ubuntu 20.04.3 LTS",
					VersionId: "20.04.3",
					Version:   "",
				},
			},
		},
	}
}
