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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	filePermission = 0o700
)

func CreateDirWithErrorCheck(t testing.TB, dirName string) {
	t.Helper()

	err := os.MkdirAll(dirName, filePermission)

	require.NoError(t, err)
}

func CreateFileWithErrorCheck(t testing.TB, dir, fileName string) *os.File {
	t.Helper()

	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func CreateCertFileWithErrorCheck(t testing.TB, dir, fileName string) *os.File {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		SerialNumber: big.NewInt(123123),
		Subject: pkix.Name{
			CommonName:   "New Subject Name",
			Organization: []string{"New Subject Org."},
		},
		Issuer: pkix.Name{
			CommonName:   "New Issuer Name",
			Organization: []string{"New Issuer Org."},
		},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})

	file := CreateFileWithErrorCheck(t, dir, "cert.pem")

	err = os.WriteFile(file.Name(), certPem, 0o600)
	require.NoError(t, err)

	return file
}

func RemoveFileWithErrorCheck(t testing.TB, fileName string) {
	t.Helper()

	err := os.Remove(fileName)

	require.NoError(t, err)
}
