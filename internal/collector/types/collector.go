// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package types

import (
	"context"

	"go.opentelemetry.io/collector/otelcol"
)

// CollectorInterface The high-level collector interface
type CollectorInterface interface {
	Run(ctx context.Context) error
	GetState() otelcol.State
	Shutdown()
}

// Ensure the original Collector struct implements your interface
var _ CollectorInterface = (*otelcol.Collector)(nil)
