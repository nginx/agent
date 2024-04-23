// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import "github.com/nginx/agent/v3/api/grpc/mpi/v1"

const configVersion = "f9a31750-566c-31b3-a763-b9fb5982547b"

func CreateConfigVersion() *v1.ConfigVersion {
	return &v1.ConfigVersion{
		Version:    configVersion,
		InstanceId: GetNginxOssInstance().GetInstanceMeta().GetInstanceId(),
	}
}
