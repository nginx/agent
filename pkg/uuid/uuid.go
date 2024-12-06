// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package uuid

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

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
	h := sha256.New()
	s := fmt.Sprintf(format, a...)
	_, _ = h.Write([]byte(s))
	id := fmt.Sprintf("%x", h.Sum(nil))

	return uuid.NewMD5(uuid.Nil, []byte(id)).String()
}

// GenerateUUIDV7 generates a UUID using the UUIDv7 standard.
//
// UUIDv7 is designed to be time-ordered and unique, making it suitable for systems requiring
// high scalability and reliable uniqueness across distributed systems.
//
// Process:
//  1. Attempts to generate a UUIDv7 using the `uuid.NewV7()` function.
//  2. If UUIDv7 generation fails, logs the error using `slog` and falls back to a SHA-256-based UUID
//     generated with the current timestamp.
//
// Returns:
//
//	A string representation of the generated UUID.
func GenerateUUIDV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		slog.Debug("issuing generating uuidv7, using sha256 and timestamp instead", "error", err)
		return Generate("%s", time.Now().String())
	}

	return id.String()
}
