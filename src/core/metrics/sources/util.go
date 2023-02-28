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

type MetricSourceLogger struct {
	once *sync.Once
}

func NewMetricSourceLogger() *MetricSourceLogger {
	return &MetricSourceLogger{&sync.Once{}}
}

func (m *MetricSourceLogger) Log(message string) {
	m.once.Do(func() {
		log.Warnf(message)
	})
	log.Tracef(message)
}
