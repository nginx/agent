/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package monitoring

import "fmt"

// WAFType denotes which type of WAF is being used
type WAFType uint

// WAFType currently can just be NAP, can be extended to support other types of WAFs if needed
const (
	NAP WAFType = iota
)

// RawLog describes the raw log entry received from the WAF
type RawLog struct {
	Origin  WAFType
	Logline string
}

// String converts WAFType enum into string
func (w WAFType) String() string {
	switch w {
	case NAP:
		return "Nginx App Protect"
	}

	return fmt.Sprintf("Unknown WAFType : %d", w)
}

// RequestStatus denotes the status of the request made to WAF
type RequestStatus uint

// RequestStatus currently can be either PASSED or BLOCKED
const (
	PASSED RequestStatus = iota
	BLOCKED
)

// String converts RequestStatus enum into string
func (r RequestStatus) String() string {
	switch r {
	case PASSED:
		return "Passed"
	case BLOCKED:
		return "Blocked"
	}

	return fmt.Sprintf("Unknown Request Status: %d", r)
}
