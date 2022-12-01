/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package grpc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppendCertKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	key, cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err.Error())
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: key,
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

	// write non-PEM data to file
	nonPEMFile := tmpDir + "/cert.nonpem"
	err = os.WriteFile(nonPEMFile, cert, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

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
			certFile: "/invalid/cert.pem",
			keyFile:  keyFile,
			isError:  true,
		},
		{
			testName: "missing key file",
			certFile: certFile,
			keyFile:  "/invalid/key.pem",
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
			tlsConfig := &tls.Config{}

			err := appendCertKeyPair(tlsConfig, tc.certFile, tc.keyFile)

			if tc.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err, err)
			}

			if tc.certAdded {
				assert.Len(t, tlsConfig.Certificates, 1, tlsConfig.Certificates)
			} else {
				assert.Len(t, tlsConfig.Certificates, 0, tlsConfig.Certificates)
			}
		})
	}

	// append to existing list test case
	t.Run("append multiple key pairs", func(t *testing.T) {
		tlsConfig := &tls.Config{}

		err := appendCertKeyPair(tlsConfig, certFile, keyFile)
		assert.Nil(t, err, err)
		err = appendCertKeyPair(tlsConfig, certFile, keyFile)
		assert.Nil(t, err, err)
		assert.Len(t, tlsConfig.Certificates, 2, tlsConfig.Certificates)
	})
}

func TestAppendRootCAs(t *testing.T) {
	tmpDir := t.TempDir()
	_, cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatal(err.Error())
	}

	caPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	caFile := tmpDir + "/ca.pem"
	err = os.WriteFile(caFile, caPem, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

	// write non-PEM data to file
	nonPEMFile := tmpDir + "/ca.nonpem"
	err = os.WriteFile(nonPEMFile, cert, 0644)
	if err != nil {
		t.Fatalf("Failed create cert file, %v", err)
	}

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
			caFile:   "/invalid/cert.pem",
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
			tlsConfig := &tls.Config{}

			err := appendRootCAs(tlsConfig, tc.caFile)

			if tc.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err, err)
			}
		})
	}
}

func generateSelfSignedCert() (keyBytes []byte, certBytes []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed generate key, %w", err)
	}
	keyBytes = x509.MarshalPKCS1PrivateKey(key)

	tmpl := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		SerialNumber: big.NewInt(123123),
		Subject: pkix.Name{
			CommonName:   "New Name",
			Organization: []string{"New Org."},
		},
		BasicConstraintsValid: true,
	}
	certBytes, err = x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cert, %w", err)
	}

	return keyBytes, certBytes, nil
}
