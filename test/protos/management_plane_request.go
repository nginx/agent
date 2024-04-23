// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import "github.com/nginx/agent/v3/api/grpc/mpi/v1"

func CreateManagementPlaneRequestConfigApplyRequest() *v1.ManagementPlaneRequest_ConfigApplyRequest {
	return &v1.ManagementPlaneRequest_ConfigApplyRequest{
		ConfigApplyRequest: &v1.ConfigApplyRequest{
			ConfigVersion: CreateConfigVersion(),
		},
	}
}
