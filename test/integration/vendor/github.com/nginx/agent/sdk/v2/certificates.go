/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func LoadCertificates(certPath, keyPath string) (*tls.Certificate, *x509.CertPool, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, nil, err
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	pool := x509.NewCertPool()
	pool.AddCert(cert.Leaf)

	return &cert, pool, nil
}

func LoadCertificate(certPath string) (*x509.Certificate, error) {
	fileContents, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	certPEMBlock, _ := pem.Decode(fileContents)
	if certPEMBlock == nil {
		return nil, fmt.Errorf("could not decode: cert was not PEM format")
	}

	cert, err := x509.ParseCertificate(certPEMBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}
