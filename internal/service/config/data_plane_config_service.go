// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	writer "github.com/nginx/agent/v3/internal/datasource/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . DataPlaneConfig
type DataPlaneConfig interface {
	ParseConfig(ctx context.Context) (any, error)
	Validate(ctx context.Context) error
	Apply(ctx context.Context) error
	Write(ctx context.Context, request *v1.ManagementPlaneRequest_ConfigApplyRequest) (
		skippedFiles writer.CacheContent, err error)
	Complete(ctx context.Context) error
	SetConfigWriter(configWriter writer.ConfigWriterInterface)
	Rollback(ctx context.Context, skippedFiles writer.CacheContent,
		request *v1.ManagementPlaneRequest_ConfigApplyRequest) error
}
