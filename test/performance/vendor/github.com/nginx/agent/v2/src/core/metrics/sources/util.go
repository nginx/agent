/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

var once = &sync.Once{}

func logMetricCollectionError(message string) {
	once.Do(func() {
		log.Warnf(message)
	})
	log.Tracef(message)
}
