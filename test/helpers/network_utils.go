// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package helpers

import (
	"fmt"
	"net"
	"testing"
	"time"

	"golang.org/x/exp/rand"
)

// GetRandomPort generates a random port for testing and checks if a port is available by attempting to bind to it
func GetRandomPort(t *testing.T) (int, error) {
	t.Helper()
	rand.Seed(uint64(time.Now().UnixNano()))

	// Define the range for dynamic ports (49152–65535 as per IANA recommendation)
	const minPort = 49152
	const maxPort = 65535

	// try up to 10 times to get a random port
	for i := 0; i < 10; i++ {
		port := rand.Intn(maxPort-minPort+1) + minPort

		if isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("could not find an available port after multiple attempts")
}

// isPortAvailable checks if a port is available by attempting to bind to it
func isPortAvailable(port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.Dial("tcp", address)
	if conn != nil {
		conn.Close()
	}

	return err != nil
}
