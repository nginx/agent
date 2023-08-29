/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"crypto/sha256"
	"fmt"
)

// GenerateID used to get the ID
func GenerateID(format string, a ...interface{}) string {
	h := sha256.New()
	s := fmt.Sprintf(format, a...)
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
