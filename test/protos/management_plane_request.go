// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

func CreateConfigApplyRequest(overview *mpi.FileOverview) *mpi.ConfigApplyRequest {
	return &mpi.ConfigApplyRequest{
		Overview:      overview,
		ConfigVersion: CreateConfigVersion(),
	}
}

func CreateManagementPlaneRequest() *mpi.ManagementPlaneRequest {
	return &mpi.ManagementPlaneRequest{
		MessageMeta: CreateMessageMeta(),
	}
}
