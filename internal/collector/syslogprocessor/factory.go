// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package syslogprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "syslog"

// NewFactory creates a factory for the syslog processor.
//
//nolint:ireturn // factory methods return interfaces by design
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		func() component.Config { return &struct{}{} },
		processor.WithLogs(createSyslogProcessor, component.StabilityLevelAlpha),
	)
}

// createSyslogProcessor instantiates the logs processor.
//
//nolint:ireturn // required to comply with component factory interface
func createSyslogProcessor(
	_ context.Context,
	settings processor.Settings,
	_ component.Config,
	next consumer.Logs,
) (processor.Logs, error) {
	settings.Logger.Info("Creating syslog processor")

	return newSyslogProcessor(next, settings), nil
}
