// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package gencert generates self-signed TLS certificates.
package tls

import (
	"encoding/pem"
	"fmt"
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:revive,gocognit
func TestGenerateSelfSignedCert(t *testing.T) {
	// Setup temp file paths
	caPath := "/tmp/test_ca.pem"
	certPath := "/tmp/test_cert.pem"
	keyPath := "/tmp/test_key.pem"
	hostNames := []string{"localhost", "::1", "127.0.0.1"}

	// Cleanup any pre-existing files from previous tests
	defer os.Remove(caPath)
	defer os.Remove(certPath)
	defer os.Remove(keyPath)

	// Define a struct for test cases
	type testCase struct {
		name          string
		caPath        string
		certPath      string
		keyPath       string
		expectedError string
		setup         func() error
		hostNames     []string
		existingCert  bool
	}

	tests := []testCase{
		{
			name: "Test 1: CA, Cert and key file exist",
			setup: func() error {
				// Ensure no cert files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				// Create valid PEM files
				keyBytes, certBytes := helpers.GenerateSelfSignedCert(t)
				caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
				certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
				keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
				if caErr := os.WriteFile(caPath, caPEM, 0o600); caErr != nil {
					return caErr
				}
				if certErr := os.WriteFile(certPath, certPEM, 0o600); certErr != nil {
					return certErr
				}

				return os.WriteFile(keyPath, keyPEM, 0o600)
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  true,
			expectedError: "",
		},
		{
			name: "Test 2: Invalid cert data",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)
				// Create dummy cert files
				if caErr := os.WriteFile(caPath, []byte("dummy ca"), 0o600); caErr != nil {
					return caErr
				}
				if certErr := os.WriteFile(certPath, []byte("dummy cert"), 0o600); certErr != nil {
					return certErr
				}

				return os.WriteFile(keyPath, []byte("dummy key"), 0o600)
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "error decoding certificate PEM block",
		},
		{
			name: "Test 3: Error writing certificate file",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				return nil
			},
			caPath:        caPath,
			certPath:      "/dev/null/cert.pem", // Path that is guaranteed to fail
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "failed to write certificate file",
		},
		{
			name: "Test 4: Error writing key file",
			setup: func() error {
				return nil
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       "/dev/null/key/pem", // Path that is guaranteed to fail
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "failed to write key file",
		},
		{
			name: "Test 5: Successful certificate generation",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				return nil
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "",
		},
		{
			name: "Test case 6: Error reading existing certificate file",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				// No read/write permissions
				if certErr := os.WriteFile(certPath, []byte("dummy cert"), 0o000); certErr != nil {
					return certErr
				}

				return os.WriteFile(keyPath, []byte("dummy key"), 0o600)
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "error reading existing certificate data",
		},
		{
			name: "Test case 7: Error reading existing key file",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				if certErr := os.WriteFile(certPath, []byte("dummy cert"), 0o600); certErr != nil {
					return certErr
				}

				return os.WriteFile(keyPath, []byte("dummy key"), 0o000)
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "error decoding certificate PEM block",
		},
		{
			name: "Test case 8: Error parsing TLS key pair",
			setup: func() error {
				// Ensure no cert or key files exist
				os.Remove(caPath)
				os.Remove(certPath)
				os.Remove(keyPath)

				// Write invalid PEM data to simulate parsing error
				err := os.WriteFile(certPath, []byte("invalid cert data"), 0o600)
				if err != nil {
					return err
				}
				err = os.WriteFile(keyPath, []byte("invalid key data"), 0o600)
				if err != nil {
					return err
				}

				return nil
			},
			caPath:        caPath,
			certPath:      certPath,
			keyPath:       keyPath,
			hostNames:     hostNames,
			existingCert:  false,
			expectedError: "error decoding certificate PEM block",
		},
	}
	// Iterate over the test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.setup()
			require.NoError(t, err)

			existingCert, genCertErr := GenerateServerCerts(tc.hostNames, tc.caPath, tc.certPath, tc.keyPath)

			// Check the results
			if tc.expectedError != "" {
				require.Error(t, genCertErr)
				assert.Contains(t, genCertErr.Error(), tc.expectedError)
			} else {
				require.NoError(t, genCertErr)
				_, err = os.Stat(tc.certPath)
				require.NoError(t, err)
				_, err = os.Stat(tc.keyPath)
				require.NoError(t, err)
			}
			fmt.Printf("tc.ExistingCert is %t\n", tc.existingCert)
			if tc.existingCert {
				require.True(t, existingCert)
			}
		})
	}
}
