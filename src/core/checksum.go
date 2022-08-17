package core

import (
	"crypto/sha256"
	"fmt"
)

// GenerateNginxID used to get the NGINX ID
func GenerateNginxID(format string, a ...interface{}) string {
	h := sha256.New()
	s := fmt.Sprintf(format, a...)
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
