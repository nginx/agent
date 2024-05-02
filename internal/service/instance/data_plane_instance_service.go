// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"github.com/nginx/agent/v3/internal/datasource/host"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . DataPlaneInstanceService
type DataPlaneInstanceService interface {
	GetInstances(ctx context.Context, processes host.NginxProcesses) []*v1.Instance
}
