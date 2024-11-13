// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package cert

import (
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyFileName        = "key.pem"
	certFileName       = "cert.pem"
	caFileName         = "ca.pem"
	nonPemCertFileName = "cert.nonpem"
	certificateType    = "CERTIFICATE"
	privateKeyType     = "RSA PRIVATE KEY"
)

func TestLoadCertificates(t *testing.T) {
	tmpDir := t.TempDir()

	key, cert := helpers.GenerateSelfSignedCert(t)

	keyContents := helpers.Cert{Name: keyFileName, Type: privateKeyType, Contents: key}
	certContents := helpers.Cert{Name: certFileName, Type: certificateType, Contents: cert}

	keyFile := helpers.WriteCertFiles(t, tmpDir, keyContents)
	certFile := helpers.WriteCertFiles(t, tmpDir, certContents)

	testCases := []struct {
		testName string
		certFile string
		keyFile  string
		isError  bool
	}{
		{
			testName: "valid files",
			certFile: certFile,
			keyFile:  keyFile,
			isError:  false,
		},
		{
			testName: "invalid cert file",
			certFile: "/invalid/cert.pem",
			keyFile:  keyFile,
			isError:  true,
		},
		{
			testName: "invalid key file",
			certFile: certFile,
			keyFile:  "/invalid/key.pem",
			isError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			certificate, pool, loadErr := LoadCertificates(tc.certFile, tc.keyFile)
			if tc.isError {
				assert.Nil(t, certificate)
				assert.Nil(t, pool)
				require.Error(t, loadErr)
			} else {
				assert.Equal(t, cert, certificate.Certificate[0])
				assert.NotNil(t, pool)
				require.NoError(t, loadErr)
			}
		})
	}
}

func TestLoadCertificate(t *testing.T) {
	tmpDir := t.TempDir()

	_, cert := helpers.GenerateSelfSignedCert(t)

	certContents := helpers.Cert{Name: certFileName, Type: certificateType, Contents: cert}
	certNonPemContents := helpers.Cert{Name: nonPemCertFileName, Type: "", Contents: cert}

	certFile := helpers.WriteCertFiles(t, tmpDir, certContents)
	nonPEMFile := helpers.WriteCertFiles(t, tmpDir, certNonPemContents)

	testCases := []struct {
		testName string
		certFile string
		isError  bool
	}{
		{
			testName: "valid cert file",
			certFile: certFile,
			isError:  false,
		},
		{
			testName: "invalid cert file",
			certFile: "/invalid/cert.pem",
			isError:  true,
		},
		{
			testName: "non-PEM cert file",
			certFile: nonPEMFile,
			isError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			certificate, loadErr := LoadCertificate(tc.certFile)
			if tc.isError {
				assert.Nil(t, certificate)
				require.Error(t, loadErr)
			} else {
				assert.Equal(t, cert, certificate.Raw)
				require.NoError(t, loadErr)
			}
		})
	}
}
