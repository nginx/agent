/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

package monitoring

import "fmt"

// WAFType denotes which type of WAF is being used
type WAFType uint

// WAFType currently can be NAPWAF
const (
	NAPWAF WAFType = iota
)

// RawLog describes the raw log entry received from the WAF
type RawLog struct {
	Origin  WAFType
	Logline string
}

// String converts WAFType enum into string
func (w WAFType) String() string {
	switch w {
	case NAPWAF:
		return "NAP WAF"
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
