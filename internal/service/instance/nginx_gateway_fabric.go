// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"fmt"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
)

type NginxGatewayFabric struct{}

func NewNginxGatewayFabric() *NginxGatewayFabric {
	return &NginxGatewayFabric{}
}

//nolint:all // remove when implemented
func (*NginxGatewayFabric) GetInstances(processes []*model.Process) ([]*instances.Instance, error) {
	return nil, fmt.Errorf("not implemented")
}
