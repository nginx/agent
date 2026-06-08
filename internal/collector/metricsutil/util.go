// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package metricsutil

// Increase calculates the delta value (difference) from monotonically increasing counters.
// If the current value is less than previous value, in the case of a reset, we take the current value.
func Increase(current, previous int64) int64 {
	if current >= previous {
		return current - previous
	}

	return current
}
