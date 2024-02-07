// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"github.com/nginx/agent/v3/api/grpc/instances"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . DataplaneConfig
type DataplaneConfig interface {
	ParseConfig(instance *instances.Instance) (any, error)
	Validate(instance *instances.Instance) error
	Reload(instance *instances.Instance) error
}
