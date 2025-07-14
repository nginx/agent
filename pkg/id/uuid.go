// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package id

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// Generate creates a UUID based on a hashed string derived from the input format and arguments.
// This function is used primarilty in generating the NGINX Id
//
// Parameters:
//   - format: A format string, similar to fmt.Sprintf.
//   - a: Variadic arguments to be substituted into the format string.
//
// Process:
//  1. Creates a SHA-256 hash from the formatted string (using `fmt.Sprintf`).
//  2. Converts the hash to a hexadecimal string.
//  3. Generates an MD5-based UUID using the hashed string.
//
// Returns:
//
//	A string representation of the generated UUID.
func Generate(format string, a ...interface{}) string {
	// need to set a default to avoid non-constant format string in call
	f := ""
	if format != "" {
		f = format
	}

	h := sha256.New()
	s := fmt.Sprintf(f, a...)
	_, _ = h.Write([]byte(s))
	id := hex.EncodeToString(h.Sum(nil))

	return uuid.NewMD5(uuid.Nil, []byte(id)).String()
}
