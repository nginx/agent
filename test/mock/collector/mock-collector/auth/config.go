// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"go.opentelemetry.io/collector/component"
)

type ServerConfig struct {
	AuthenticatorID string `mapstructure:"authenticator"`
}

type Config struct {
	AuthenticatorID component.ID `mapstructure:",squash"`
}

//nolint:ireturn
func CreateDefaultConfig() component.Config {
	return &Config{
		AuthenticatorID: HeadersCheckID,
	}
}
