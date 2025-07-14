// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package auth

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	auth "go.opentelemetry.io/collector/extension/extensionauth"
	"go.uber.org/zap"
)

const (
	AuthenticatorName = "headers_check"
)

var (
	aType          = component.MustNewType(AuthenticatorName)
	HeadersCheckID = component.MustNewID(AuthenticatorName)
)

type HeadersCheck struct {
	logger          *zap.SugaredLogger
	AuthenticatorID component.ID `mapstructure:"authenticator"`
}

type Option func(*HeadersCheck)

// Ensure that the authenticator implements the auth.Server interface.
var _ auth.Server = (*HeadersCheck)(nil)

//nolint:ireturn
func NewFactory() extension.Factory {
	return extension.NewFactory(
		aType,
		CreateDefaultConfig,
		CreateAuthExtensionFunc,
		component.StabilityLevelBeta,
	)
}

func (a *HeadersCheck) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (a *HeadersCheck) Shutdown(_ context.Context) error {
	return nil
}

func (a *HeadersCheck) Authenticate(ctx context.Context, headers map[string][]string) (context.Context, error) {
	a.logger.Info("Headers", zap.Any("headers", headers))
	return ctx, nil
}

//nolint:ireturn
func CreateAuthExtensionFunc(
	_ context.Context,
	setting extension.Settings,
	_ component.Config,
) (extension.Extension, error) {
	logger := setting.Logger.Sugar()

	a := &HeadersCheck{
		AuthenticatorID: setting.ID,
		logger:          logger,
	}

	return a, nil
}
