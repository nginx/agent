/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

package collector

import (
	"github.com/sirupsen/logrus"
)

// NAPWAFConfig holds the config for NAPWAFConfig Collector.
type NAPWAFConfig struct {
	SyslogIP   string
	SyslogPort int
	Logger     *logrus.Entry
}
