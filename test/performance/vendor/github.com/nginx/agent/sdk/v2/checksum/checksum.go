/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package checksum

import (
	"crypto/sha256"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"
)

// Checksum - calculate checksum from []byte
func Checksum(b []byte) string {
	h := sha256.New()
	_, _ = h.Write(b)
	return string(h.Sum(nil))
}

func HexChecksum(b []byte) string {
	return fmt.Sprintf("%x", Checksum(b))
}

// Chunk - split bytes to chunk limits
func Chunk(buf []byte, lim int) [][]byte {
	var chunk []byte
	bufSize := len(buf)

	if bufSize == 0 {
		return [][]byte{}
	}

	if bufSize <= lim {
		return [][]byte{buf}
	}

	chuckSize := bufSize / lim

	if chuckSize > math.MaxInt64-1 {
		log.Error("Unable to chuck payload. Data too large.")
		return [][]byte{}
	}

	chunks := make([][]byte, 0, chuckSize+1)

	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}

	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}

	return chunks
}
