/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collector

import (
	"context"
	"sync"

	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

// Collector is the interface implemented by collectors who wish to
// collect Raw Log data from WAF Instances.
type Collector interface {
	// Collect starts collecting on collect chan until ctx.Done() chan gets a signal
	Collect(ctx context.Context, wg *sync.WaitGroup, collect chan<- *monitoring.RawLog)
}
