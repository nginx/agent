// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package gencert generates self-signed TLS certificates.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"time"
)

const (
	caOrganization      = "F5 Inc. CA"
	certOrganization    = "F5 Inc."
	certFilePermissions = 0o600
	keyFilePermissions  = 0o600
)

type certReq struct {
	template   *x509.Certificate
	parent     *x509.Certificate
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

func genCert(req *certReq) (*x509.Certificate, []byte) {
	certBytes, createCertErr := x509.CreateCertificate(
		rand.Reader,
		req.template,
		req.parent,
		req.publicKey,
		req.privateKey,
	)

	if createCertErr != nil {
		slog.Error("Failed to generate certificate", "error", createCertErr)
		return &x509.Certificate{}, []byte{}
	}

	cert, parseCertErr := x509.ParseCertificate(certBytes)
	if parseCertErr != nil {
		slog.Error("Failed to parse certificate")
		return &x509.Certificate{}, []byte{}
	}

	b := pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
	certPEM := pem.EncodeToMemory(&b)

	return cert, certPEM
}

func GenerateCA(now time.Time, caCertPath string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	// Generate key pair for the CA
	caKeyPair, caKeyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if caKeyErr != nil {
		return &x509.Certificate{}, &ecdsa.PrivateKey{}, fmt.Errorf("failed to generate CA private key: %w", caKeyErr)
	}

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{certOrganization}},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.AddDate(1, 0, 0), // 1 year
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// CA is self signed
	caRequest := certReq{
		template:   &caTemplate,
		parent:     &caTemplate,
		publicKey:  &caKeyPair.PublicKey,
		privateKey: caKeyPair,
	}

	caCert, caCertPEM := genCert(&caRequest)
	if len(caCertPEM) == 0 {
		slog.Error("Error generating certificate authority")
	}

	// Write the CA certificate to a file
	slog.Debug("About to write CA file", "path", caCertPath)
	writeCAErr := os.WriteFile(caCertPath, caCertPEM, certFilePermissions)
	if writeCAErr != nil {
		return &x509.Certificate{}, &ecdsa.PrivateKey{}, fmt.Errorf(
			"failed to write ca file: %w",
			writeCAErr,
		)
	}

	return caCert, caKeyPair, nil
}

// nolint: revive
func GenerateServerCert(hostnames []string, caPath, certPath, keyPath string) error {
	// Check for and return existing cert if it already exists
	existingCertErr := ReturnExistingCert(certPath)
	if existingCertErr != nil {
		return fmt.Errorf("error reading existing certificate data: %w", existingCertErr)
	}

	// Get the local time zone
	locationCurrentzone, locErr := time.LoadLocation("Local")
	if locErr != nil {
		return fmt.Errorf("error detecting local timezone: %w", locErr)
	}
	now := time.Now().In(locationCurrentzone)

	// Create CA first
	caCert, caKeyPair, caErr := GenerateCA(now, caPath)
	if caErr != nil {
		return fmt.Errorf("error generating certificate authority: %w", caErr)
	}

	// Generate key pair for the server certficate
	servKeyPair, servKeyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if servKeyErr != nil {
		return fmt.Errorf("failed to generate server keypair: %w", servKeyErr)
	}

	servTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{caOrganization},
		},
		NotBefore:   now.Add(-time.Minute),
		NotAfter:    now.AddDate(1, 0, 0), // 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:    hostnames,
	}

	servRequest := certReq{
		template:   &servTemplate,
		parent:     caCert,
		publicKey:  &servKeyPair.PublicKey,
		privateKey: caKeyPair,
	}

	// Generate server certficated signed by the CA
	_, servCertPEM := genCert(&servRequest)
	if len(servCertPEM) == 0 {
		return errors.New("error generating server certificate")
	}

	// Write the certificate to a file
	writeCertErr := os.WriteFile(certPath, servCertPEM, certFilePermissions)
	if writeCertErr != nil {
		return fmt.Errorf("failed to write certificate file: %w", writeCertErr)
	}

	// Write the private key to a file
	servKeyBytes, marshalErr := x509.MarshalECPrivateKey(servKeyPair)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal private key file: %w", marshalErr)
	}
	b := pem.Block{Type: "EC PRIVATE KEY", Bytes: servKeyBytes}
	servKeyPEM := pem.EncodeToMemory(&b)
	writeKeyErr := os.WriteFile(keyPath, servKeyPEM, keyFilePermissions)
	if writeKeyErr != nil {
		return fmt.Errorf("failed to write key file: %w", writeKeyErr)
	}

	return nil
}

func ReturnExistingCert(certPath string) error {
	if _, certErr := os.Stat(certPath); certErr == nil {
		certBytes, certReadErr := os.ReadFile(certPath)
		if certReadErr != nil {
			return fmt.Errorf("error reading existing certificate file")
		}
		certPEM, _ := pem.Decode(certBytes)
		if certPEM == nil {
			return errors.New("error decoding certificate PEM block")
		}
		_, parseErr := x509.ParseCertificate(certPEM.Bytes)
		if parseErr == nil {
			slog.Warn("Certificate file already exists, skipping self-signed certificate generation")
			return nil
		}

		return fmt.Errorf("error parsing existing certificate: %w", parseErr)
	}

	return nil
}
