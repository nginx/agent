// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package securityviolationsfilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "securityviolationsfilter"

// NewFactory creates a factory for the security-violations-filter processor.
//
//nolint:ireturn // factory methods return interfaces by design
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		func() component.Config { return &struct{}{} },
		processor.WithLogs(createLogsProcessor, component.StabilityLevelAlpha),
	)
}

// createLogsProcessor instantiates the security-violations-filter logs processor.
//
//nolint:ireturn // required to comply with component factory interface
func createLogsProcessor(
	_ context.Context,
	settings processor.Settings,
	_ component.Config,
	next consumer.Logs,
) (processor.Logs, error) {
	settings.Logger.Info("Creating security violations filter processor")

	return newSecurityViolationsFilterProcessor(next, settings), nil
}
