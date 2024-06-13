// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
//

package helpers

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Useful functions pulled in from non public
// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
func GetAvailablePort(t testing.TB) int {
	t.Helper()
	endpoint := GetAvailableLocalAddress(t)
	_, port, err := net.SplitHostPort(endpoint)
	require.NoError(t, err)

	portInt, err := strconv.Atoi(port)
	require.NoError(t, err)

	return portInt
}

// GetAvailableLocalAddress finds an available local port on tcp network and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalAddress(t testing.TB) string {
	t.Helper()

	return GetAvailableLocalNetworkAddress(t, "tcp")
}

// GetAvailableLocalNetworkAddress finds an available local port on specified network and returns an endpoint
// describing it. The port is available for opening when this function returns
// provided that there is no race by some other code to grab the same port
// immediately.
func GetAvailableLocalNetworkAddress(t testing.TB, network string) string {
	t.Helper()
	// Retry has been added for windows as net.Listen can return a port that is not actually available. Details can be
	// found in https://github.com/docker/for-win/issues/3171 but to summarize Hyper-V will reserve ranges of ports
	// which do not show up under the "netstat -ano" but can only be found by
	// "netsh interface ipv4 show excludedportrange protocol=tcp".  We'll use []exclusions to hold those ranges and
	// retry if the port returned by GetAvailableLocalAddress falls in one of those them.

	portFound := false

	var endpoint string
	for !portFound {
		endpoint = findAvailableAddress(t, network)
		_, _, err := net.SplitHostPort(endpoint)
		require.NoError(t, err)
		portFound = true
	}

	return endpoint
}

func findAvailableAddress(t testing.TB, network string) string {
	t.Helper()

	switch network {
	// net.Listen supported network strings
	case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		ln, err := net.Listen(network, "localhost:0")
		require.NoError(t, err, "Failed to get a free local port")
		// There is a possible race if something else takes this same port before
		// the test uses it, however, that is unlikely in practice.
		defer func() {
			assert.NoError(t, ln.Close())
		}()

		return ln.Addr().String()
	// net.ListenPacket supported network strings
	case "udp", "udp4", "udp6", "unixgram":
		ln, err := net.ListenPacket(network, "localhost:0")
		require.NoError(t, err, "Failed to get a free local port")
		// There is a possible race if something else takes this same port before
		// the test uses it, however, that is unlikely in practice.
		defer func() {
			assert.NoError(t, ln.Close())
		}()

		return ln.LocalAddr().String()
	}

	return ""
}
