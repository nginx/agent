// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package client

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ConfigClient

type ConfigClient interface {
	GetFilesMetadata(ctx context.Context, request *v1.GetOverviewRequest) (*v1.FileOverview, error)
	GetFile(ctx context.Context, request *v1.GetFileRequest) (*v1.FileContents, error)
}
