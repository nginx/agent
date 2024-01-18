/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package datasource

import (
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model/os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_datasource.go . Datasource
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource mock_datasource.go | sed -e s\\/datasource\\\\.\\/\\/g > mock_datasource_fixed.go"
//go:generate mv mock_datasource_fixed.go mock_datasource.go
type Datasource interface {
	GetInstances(processes []*os.Process) ([]*instances.Instance, error)
}
