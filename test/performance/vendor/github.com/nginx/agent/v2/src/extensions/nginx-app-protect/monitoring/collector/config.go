/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collector

import (
	"github.com/sirupsen/logrus"
)

// NAPConfig holds the config for NAPConfig Collector.
type NAPConfig struct {
	SyslogIP   string
	SyslogPort int
	Logger     *logrus.Entry
}
