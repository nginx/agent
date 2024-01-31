/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"fmt"

	"github.com/nginx/agent/v3/api/grpc/instances"
)

type NginxGatewayFabric struct{}

func NewNginxGatewayFabric() *NginxGatewayFabric {
	return &NginxGatewayFabric{}
}

func (*NginxGatewayFabric) ParseConfig(instance *instances.Instance) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) Validate() error {
	return fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) Reload() error {
	return fmt.Errorf("not implemented")
}
