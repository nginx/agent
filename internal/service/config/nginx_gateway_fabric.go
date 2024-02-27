// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"fmt"
	"log/slog"

	writer "github.com/nginx/agent/v3/internal/datasource/config"
)

type NginxGatewayFabric struct{}

func NewNginxGatewayFabric() *NginxGatewayFabric {
	return &NginxGatewayFabric{}
}

func (*NginxGatewayFabric) ParseConfig() (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) SetConfigContext(_ any) {}

func (*NginxGatewayFabric) Validate() error {
	return fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) Apply() error {
	return fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) Complete() error {
	return fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) Write(_ context.Context, _, _ string) (skippedFiles map[string]struct{}, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (*NginxGatewayFabric) SetConfigWriter(configWriter writer.ConfigWriterInterface) {
	slog.Warn("not implemented")
}
