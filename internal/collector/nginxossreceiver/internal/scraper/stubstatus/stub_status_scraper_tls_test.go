// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stubstatus

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
)

func TestStubStatusScraperTLS(t *testing.T) {
	// Create a test CA certificate and key
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NGINX Agent Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)

	// Create a test server certificate signed by the CA
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"NGINX Agent Test"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Save CA certificate to a file
	caFile := filepath.Join(tempDir, "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	err = os.WriteFile(caFile, caPEM, 0o600)
	require.NoError(t, err)

	// Create a TLS config for the server
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{certBytes},
				PrivateKey:  certPrivKey,
			},
		},
	}

	// Create a test server with TLS
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

	// Test with TLS configuration
	t.Run("with TLS CA", func(t *testing.T) {
		cfg, ok := config.CreateDefaultConfig().(*config.Config)
		require.True(t, ok)

		cfg.APIDetails.URL = server.URL + "/status"
		cfg.APIDetails.Ca = caFile

		scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

		err := scraper.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)

		_, err = scraper.Scrape(context.Background())
		assert.NoError(t, err)
	})
}

func TestStubStatusScraperUnixSocket(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "TestStubStatusScraperUnixSocket")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	socketPath := filepath.Join(tempDir, "nginx.sock")

	// Create a Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	// Start a simple HTTP server on the Unix socket
	server := &http.Server{
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
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
		}),
	}

	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	// Test with Unix socket
	t.Run("with Unix socket", func(t *testing.T) {
		cfg, ok := config.CreateDefaultConfig().(*config.Config)
		require.True(t, ok)

		cfg.APIDetails.URL = "http://unix/status"
		cfg.APIDetails.Listen = "unix:" + socketPath

		scraper := NewScraper(receivertest.NewNopSettings(component.Type{}), cfg)

		err := scraper.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)

		_, err = scraper.Scrape(context.Background())
		assert.NoError(t, err)
	})
}
