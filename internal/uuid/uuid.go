// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package uuid

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
)

func Generate(format string, a ...interface{}) string {
	h := sha256.New()
	s := fmt.Sprintf(format, a...)
	_, _ = h.Write([]byte(s))
	id := fmt.Sprintf("%x", h.Sum(nil))

	return uuid.NewMD5(uuid.Nil, []byte(id)).String()
}
