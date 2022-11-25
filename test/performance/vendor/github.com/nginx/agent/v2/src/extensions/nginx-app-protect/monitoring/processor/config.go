/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processor

import (
	"github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core/metrics"
)

// Config holds the config for Processor.
type Config struct {
	Logger     *logrus.Entry
	Workers    int
	CommonDims *metrics.CommonDim
}
