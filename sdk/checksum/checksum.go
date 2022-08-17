package checksum

import (
	"crypto/sha256"
	"fmt"
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
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}
	return chunks
}
