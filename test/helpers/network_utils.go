// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package helpers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"testing"
)

// RandomPort generates a random port for testing and checks if a port is available by attempting to bind to it
func RandomPort(t *testing.T, ctx context.Context) (int, error) {
	t.Helper()

	// Define the range for dynamic ports (49152–65535 as per IANA recommendation)
	const minPort = 49152
	const maxPort = 65535

	// try up to 10 times to get a random port
	for range 10 {
		maxValue := &big.Int{}
		maxValue.SetInt64(maxPort - minPort + 1)

		port, err := rand.Int(rand.Reader, maxValue)
		if err != nil {
			return 0, err
		}

		portNumber := int(port.Int64()) + minPort

		if isPortAvailable(ctx, portNumber) {
			return portNumber, nil
		}
	}

	return 0, errors.New("could not find an available port after multiple attempts")
}

// isPortAvailable checks if a port is available by attempting to bind to it
func isPortAvailable(ctx context.Context, port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if conn != nil {
		conn.Close()
	}

	return err != nil
}
