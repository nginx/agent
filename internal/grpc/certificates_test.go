// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package grpc

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyFileName        = "key.pem"
	certFileName       = "cert.pem"
	caFileName         = "ca.pem"
	nonPemCaFileName   = "ca.nonpem"
	nonPemCertFileName = "cert.nonpem"
	certificateType    = "CERTIFICATE"
	privateKeyType     = "RSA PRIVATE KEY"
)

var pathSeparator = string(os.PathSeparator)

func TestAppendCertKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	key, cert := helpers.GenerateSelfSignedCert(t)

	keyContents := helpers.Cert{Name: keyFileName, Type: privateKeyType, Contents: key}
	certContents := helpers.Cert{Name: certFileName, Type: certificateType, Contents: cert}
	certNonPemContents := helpers.Cert{Name: nonPemCertFileName, Type: "", Contents: cert}

	keyFile := helpers.WriteCertFiles(t, tmpDir, keyContents)
	certFile := helpers.WriteCertFiles(t, tmpDir, certContents)
	nonPEMFile := helpers.WriteCertFiles(t, tmpDir, certNonPemContents)

	testCases := []struct {
		testName  string
		certFile  string
		keyFile   string
		isError   bool
		certAdded bool
	}{
		{
			testName:  "valid files",
			certFile:  certFile,
			keyFile:   keyFile,
			isError:   false,
			certAdded: true,
		},
		{
			testName: "no files",
			certFile: "",
			keyFile:  "",
			isError:  false,
		},
		{
			testName: "only cert file",
			certFile: certFile,
			keyFile:  "",
			isError:  true,
		},
		{
			testName: "only key file",
			certFile: certFile,
			keyFile:  "",
			isError:  true,
		},
		{
			testName: "missing cert file",
			certFile: fmt.Sprintf("%s%s%s%s", pathSeparator, "invalid", pathSeparator, certFileName),
			keyFile:  keyFile,
			isError:  true,
		},
		{
			testName: "missing key file",
			certFile: certFile,
			keyFile:  fmt.Sprintf("%s%s%s%s", pathSeparator, "invalid", pathSeparator, keyFileName),
			isError:  true,
		},
		{
			testName: "non PEM format cert",
			certFile: nonPEMFile,
			keyFile:  keyFile,
			isError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}

			err := appendCertKeyPair(tlsConfig, tc.certFile, tc.keyFile)

			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.certAdded {
				assert.Len(t, tlsConfig.Certificates, 1, tlsConfig.Certificates)
			} else {
				assert.Empty(t, tlsConfig.Certificates)
			}
		})
	}

	// append to existing list test case
	t.Run("append multiple key pairs", func(t *testing.T) {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		err := appendCertKeyPair(tlsConfig, certFile, keyFile)
		require.NoError(t, err)

		err = appendCertKeyPair(tlsConfig, certFile, keyFile)
		require.NoError(t, err)

		assert.Len(t, tlsConfig.Certificates, 2, tlsConfig.Certificates)
	})
}

func TestAppendRootCAs(t *testing.T) {
	tmpDir := t.TempDir()
	_, cert := helpers.GenerateSelfSignedCert(t)

	caContents := helpers.Cert{Name: caFileName, Type: certificateType, Contents: cert}
	certNonPemContents := helpers.Cert{Name: nonPemCaFileName, Type: "", Contents: cert}

	caFile := helpers.WriteCertFiles(t, tmpDir, caContents)
	nonPEMFile := helpers.WriteCertFiles(t, tmpDir, certNonPemContents)

	testCases := []struct {
		testName string
		caFile   string
		isError  bool
	}{
		{
			testName: "valid ca cert",
			caFile:   caFile,
			isError:  false,
		},
		{
			testName: "no file",
			caFile:   "",
			isError:  false,
		},
		{
			testName: "missing ca file",
			caFile:   fmt.Sprintf("%s%s%s%s", pathSeparator, "invalid", pathSeparator, certFileName),
			isError:  true,
		},
		{
			testName: "non PEM format cert",
			caFile:   nonPEMFile,
			isError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}

			err := appendRootCAs(tlsConfig, tc.caFile)

			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
