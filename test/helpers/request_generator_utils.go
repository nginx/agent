// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
)

func CreateManagementPlaneRequestConfigApplyRequest() *v1.ManagementPlaneRequest_ConfigApplyRequest {
	return &v1.ManagementPlaneRequest_ConfigApplyRequest{
		ConfigApplyRequest: &v1.ConfigApplyRequest{
			ConfigVersion: &v1.ConfigVersion{
				Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
				InstanceId: protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId(),
			},
		},
	}
}
