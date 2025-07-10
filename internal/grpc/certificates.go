// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// appendRootCAs will read, parse, and append any certificates found in the
// file at caFile to the RootCAs in the provided tls Config. If no filepath
// is provided the tls Config is unmodified. By default, if there are no RootCAs
// in the tls Config, it will automatically use the host OS's CA pool.
func appendRootCAs(tlsConfig *tls.Config, caFile string) error {
	if caFile == "" {
		return nil
	}

	ca, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("read CA file (%s): %w", caFile, err)
	}

	// If CAs have already been set, append to existing
	caPool := tlsConfig.RootCAs
	if caPool == nil {
		caPool = x509.NewCertPool()
	}

	if !caPool.AppendCertsFromPEM(ca) {
		return fmt.Errorf("parse CA cert (%s)", caFile)
	}

	tlsConfig.RootCAs = caPool

	return nil
}

// appendCertKeyPair will attempt to load a cert and key pair from the provided
// file paths and append to the Certificates list in the provided tls Config. If
// no files are provided the tls Config is unmodified. If only one file (key or
// cert) is provided, an error is produced.
func appendCertKeyPair(tlsConfig *tls.Config, certFile, keyFile string) error {
	if certFile == "" && keyFile == "" {
		return nil
	}
	if certFile == "" || keyFile == "" {
		return errors.New("cert and key must both be provided")
	}

	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("load X509 keypair: %w", err)
	}

	tlsConfig.Certificates = append(tlsConfig.Certificates, certificate)

	return nil
}
