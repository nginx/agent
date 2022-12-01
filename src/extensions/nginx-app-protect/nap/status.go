/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

// Enums for Status
const (
	UNDEFINED Status = iota
	MISSING
	INSTALLED
	RUNNING
)

// String get the string representation of the enum
func (s Status) String() string {
	switch s {
	case MISSING:
		return "missing"
	case INSTALLED:
		return "installed"
	case RUNNING:
		return "running"
	}
	return "unknown"
}
