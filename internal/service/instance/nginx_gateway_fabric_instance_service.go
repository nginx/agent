// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
)

type NginxGatewayFabric struct{}

func NewNginxGatewayFabric() *NginxGatewayFabric {
	return &NginxGatewayFabric{}
}

//nolint:all // remove when implemented
func (*NginxGatewayFabric) GetInstances(_ context.Context, _ []*model.Process) []*v1.Instance {
	return make([]*v1.Instance, 0)
}
