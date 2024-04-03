// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type InstanceConfigUpdateRequest struct {
	Instance *v1.Instance
	Location string
}
