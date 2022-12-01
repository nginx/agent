/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed generate key, %v", err)
	}

	tml := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		SerialNumber: big.NewInt(123123),
		Subject: pkix.Name{
			CommonName:   "New Name",
			Organization: []string{"New Org."},
		},
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed create cert, %v", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	certFile := tmpDir + "/cert.pem"
	err = os.WriteFile(certFile, certPem, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

	keyFile := tmpDir + "/key.pem"
	err = os.WriteFile(keyFile, keyPem, 0640)
	if err != nil {
		t.Fatalf("Failed create key file, %v", err)
	}

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
			certificate, pool, err := LoadCertificates(tc.certFile, tc.keyFile)
			if tc.isError {
				assert.Nil(t, certificate)
				assert.Nil(t, pool)
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, cert, certificate.Certificate[0])
				assert.NotNil(t, pool)
				assert.Nil(t, err)
			}
		})
	}
}

func TestLoadCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed generate key, %v", err)
	}

	tml := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		SerialNumber: big.NewInt(123123),
		Subject: pkix.Name{
			CommonName:   "New Name",
			Organization: []string{"New Org."},
		},
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed create cert, %v", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	// write valid PEM certificate to file
	certFile := tmpDir + "/cert.pem"
	err = os.WriteFile(certFile, certPem, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

	// write non-PEM data to file
	nonPEMFile := tmpDir + "/cert.nonpem"
	err = os.WriteFile(nonPEMFile, cert, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

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
			certificate, err := LoadCertificate(tc.certFile)
			if tc.isError {
				assert.Nil(t, certificate)
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, cert, certificate.Raw)
				assert.Nil(t, err)
			}
		})
	}
}
