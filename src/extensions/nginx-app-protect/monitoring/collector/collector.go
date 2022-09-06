/*
 * Copyright (C) F5 Networks, Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Networks, Inc.
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
