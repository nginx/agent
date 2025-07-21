// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Cert struct {
	Name     string
	Type     string
	Contents []byte
}

const (
	permission          = 0o600
	serialNumber        = 123123
	years, months, days = 5, 0, 0
	bits                = 2048
)

func GenerateSelfSignedCert(t testing.TB) (keyBytes, certBytes []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Errorf("failed generate key, %s", err)
		t.Fail()
	}
	keyBytes = x509.MarshalPKCS1PrivateKey(key)

	tmpl := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(years, months, days),
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			CommonName:   "New Name",
			Organization: []string{"New Org."},
		},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}
	certBytes, err = x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Errorf("failed to create cert, %s", err)
		t.Fail()
	}

	return keyBytes, certBytes
}

func WriteCertFiles(t *testing.T, location string, cert Cert) string {
	t.Helper()
	pemContents := pem.EncodeToMemory(&pem.Block{
		Type:  cert.Type,
		Bytes: cert.Contents,
	})

	certFile := path.Join(location, cert.Name)

	err := os.WriteFile(certFile, pemContents, permission)
	require.NoError(t, err)

	return certFile
}
