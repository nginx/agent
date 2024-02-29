// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"

	writer "github.com/nginx/agent/v3/internal/datasource/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . DataPlaneConfig
type DataPlaneConfig interface {
	ParseConfig() (any, error)
	Validate() error
	Apply() error
	Write(ctx context.Context, filesURL, tenantID string) (skippedFiles writer.CacheContent, err error)
	Complete() error
	SetConfigWriter(configWriter writer.ConfigWriterInterface)
	Rollback(ctx context.Context, skippedFiles writer.CacheContent, filesURL, tenantID, instanceID string) error
}
