// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stubstatus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/test/helpers"
)

func TestStubStatusScraperTLS(t *testing.T) {
	// Generate self-signed certificate using helper
	keyBytes, certBytes := helpers.GenerateSelfSignedCert(t)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Save certificate to a file
	certFile := helpers.WriteCertFiles(t, tempDir, helpers.Cert{
		Name:     "server.crt",
		Type:     "CERTIFICATE",
		Contents: certBytes,
	})

	// Parse the private key
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	require.NoError(t, err)

	// Create a TLS config with our self-signed certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  key,
	}

	serverTLSConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{tlsCert},
	}

	// Create a test server with our custom TLS config
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/status" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`Active connections: 291
server accepts handled requests
 16630948 16630946 31070465
Reading: 6 Writing: 179 Waiting: 106
`))

			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))

	server.TLS = serverTLSConfig
	server.StartTLS()
	defer server.Close()

	// Test with TLS configuration using our self-signed certificate
	t.Run("Test 1: self-signed TLS", func(t *testing.T) {
		cfg, ok := config.CreateDefaultConfig().(*config.Config)
		require.True(t, ok)

		cfg.APIDetails.URL = server.URL + "/status"
		// Use the self-signed certificate for verification
		cfg.APIDetails.Ca = certFile

		scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

		startErr := scraper.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, startErr)

		_, err = scraper.Scrape(context.Background())
		assert.NoError(t, err, "Scraping with self-signed certificate should succeed")
	})
}

func TestStubStatusScraperUnixSocket(t *testing.T) {
	// Create a test server with a Unix domain socket
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/status" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`Active connections: 291
server accepts handled requests
 16630948 16630946 31070465
Reading: 6 Writing: 179 Waiting: 106
`))

			return
		}
		rw.WriteHeader(http.StatusNotFound)
	})

	// Create a socket file in a temporary directory with a shorter path
	socketPath := "/tmp/nginx-test.sock"

	// Clean up any existing socket file
	os.Remove(socketPath)

	// Create a listener for the Unix socket
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err, "Failed to create Unix socket listener")

	// Create a test server with our custom listener
	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}

	// Start the server
	server.Start()

	// Ensure cleanup of the socket file
	t.Cleanup(func() {
		server.Close()
		os.Remove(socketPath)
	})

	// Test with Unix socket
	t.Run("Test 1: Unix socket", func(t *testing.T) {
		cfg, ok := config.CreateDefaultConfig().(*config.Config)
		require.True(t, ok)

		cfg.APIDetails.URL = "http://unix/status"
		cfg.APIDetails.Listen = "unix:" + socketPath

		scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

		startErr := scraper.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, startErr)

		_, err = scraper.Scrape(context.Background())
		assert.NoError(t, err)
	})
}
