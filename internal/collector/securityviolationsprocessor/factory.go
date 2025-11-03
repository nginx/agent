// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "securityviolations"

// NewFactory creates a factory for the securityviolations processor.
//
//nolint:ireturn // factory methods return interfaces by design
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		func() component.Config { return &struct{}{} },
		processor.WithLogs(createSecurityViolationsProcessor, component.StabilityLevelAlpha),
	)
}

// createSecurityViolationsProcessor instantiates the logs processor.
//
//nolint:ireturn // required to comply with component factory interface
func createSecurityViolationsProcessor(
	_ context.Context,
	settings processor.Settings,
	_ component.Config,
	next consumer.Logs,
) (processor.Logs, error) {
	settings.Logger.Info("Creating security violations processor")

	return newSecurityViolationsProcessor(next, settings), nil
}
