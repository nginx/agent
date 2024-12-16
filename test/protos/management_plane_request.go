// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

func CreateConfigApplyRequest(overview *mpi.FileOverview) *mpi.ConfigApplyRequest {
	return &mpi.ConfigApplyRequest{
		Overview: overview,
	}
}

func CreateManagementPlaneRequest() *mpi.ManagementPlaneRequest {
	return &mpi.ManagementPlaneRequest{
		MessageMeta: CreateMessageMeta(),
	}
}

func CreatAPIActionRequestNginxPlusGetHTTPServers(upstream, instanceID string) *mpi.ManagementPlaneRequest {
	return &mpi.ManagementPlaneRequest{
		MessageMeta: CreateMessageMeta(),
		Request: &mpi.ManagementPlaneRequest_ActionRequest{
			ActionRequest: &mpi.APIActionRequest{
				InstanceId: instanceID,
				Action: &mpi.APIActionRequest_NginxPlusAction{
					NginxPlusAction: &mpi.NGINXPlusAction{
						Action: &mpi.NGINXPlusAction_GetHttpUpstreamServers{
							GetHttpUpstreamServers: &mpi.GetHTTPUpstreamServers{
								HttpUpstreamName: upstream,
							},
						},
					},
				},
			},
		},
	}
}
