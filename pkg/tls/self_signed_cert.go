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

// Returns x509 Certificate object and bytes in PEM format
func genCert(req *certReq) (*x509.Certificate, []byte, error) {
	certBytes, createCertErr := x509.CreateCertificate(
		rand.Reader,
		req.template,
		req.parent,
		req.publicKey,
		req.privateKey,
	)

	if createCertErr != nil {
		return &x509.Certificate{}, []byte{}, fmt.Errorf("error generating certificate: %w", createCertErr)
	}

	cert, parseCertErr := x509.ParseCertificate(certBytes)
	if parseCertErr != nil {
		return &x509.Certificate{}, []byte{}, fmt.Errorf("error parsing certificate: %w", parseCertErr)
	}

	b := pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
	certPEM := pem.EncodeToMemory(&b)

	return cert, certPEM, nil
}

// Generates a CA, returns x509 Certificate and private key for signing server certificates
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

	caCert, caCertPEM, caErr := genCert(&caRequest)
	if caErr != nil {
		return &x509.Certificate{}, &ecdsa.PrivateKey{}, fmt.Errorf(
			"error generating certificate authority: %w",
			caErr)
	}

	// Write the CA certificate to a file
	writeCAErr := os.WriteFile(caCertPath, caCertPEM, certFilePermissions)
	if writeCAErr != nil {
		return &x509.Certificate{}, &ecdsa.PrivateKey{}, fmt.Errorf(
			"failed to write ca file: %w",
			writeCAErr,
		)
	}

	return caCert, caKeyPair, nil
}

// Writes CA, Cert, Key to specified destinations. If cert files are already present, does nothing, returns true
// nolint: revive
func GenerateServerCerts(hostnames []string, caPath, certPath, keyPath string) (existingCert bool, err error) {
	// Check for and return existing cert if it already exists
	existingCert, existingCertErr := DoesCertAlreadyExist(certPath)
	if existingCertErr != nil {
		return false, fmt.Errorf("error reading existing certificate data: %w", existingCertErr)
	}
	if existingCert {
		return true, nil
	}

	// Get the local time zone
	locationCurrentzone, locErr := time.LoadLocation("Local")
	if locErr != nil {
		return false, fmt.Errorf("error detecting local timezone: %w", locErr)
	}
	now := time.Now().In(locationCurrentzone)

	// Create CA first
	caCert, caKeyPair, caErr := GenerateCA(now, caPath)
	if caErr != nil {
		return false, fmt.Errorf("error generating certificate authority: %w", caErr)
	}

	// Generate key pair for the server certficate
	servKeyPair, servKeyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if servKeyErr != nil {
		return false, fmt.Errorf("failed to generate server keypair: %w", servKeyErr)
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
	_, servCertPEM, servCertErr := genCert(&servRequest)
	if servCertErr != nil {
		return false, fmt.Errorf("error generating server certificate: %w", servCertErr)
	}

	// Write the certificate to a file
	writeCertErr := os.WriteFile(certPath, servCertPEM, certFilePermissions)
	if writeCertErr != nil {
		return false, fmt.Errorf("failed to write certificate file: %w", writeCertErr)
	}

	// Write the private key to a file
	servKeyBytes, marshalErr := x509.MarshalECPrivateKey(servKeyPair)
	if marshalErr != nil {
		return false, fmt.Errorf("failed to marshal private key file: %w", marshalErr)
	}
	b := pem.Block{Type: "EC PRIVATE KEY", Bytes: servKeyBytes}
	servKeyPEM := pem.EncodeToMemory(&b)
	writeKeyErr := os.WriteFile(keyPath, servKeyPEM, keyFilePermissions)
	if writeKeyErr != nil {
		return false, fmt.Errorf("failed to write key file: %w", writeKeyErr)
	}

	return false, nil
}

// Returns true if a valid certificate is found at certPath
func DoesCertAlreadyExist(certPath string) (bool, error) {
	if _, certErr := os.Stat(certPath); certErr == nil {
		certBytes, certReadErr := os.ReadFile(certPath)
		if certReadErr != nil {
			return false, fmt.Errorf("error reading existing certificate file")
		}
		certPEM, _ := pem.Decode(certBytes)
		if certPEM == nil {
			return false, errors.New("error decoding certificate PEM block")
		}
		_, parseErr := x509.ParseCertificate(certPEM.Bytes)
		if parseErr == nil {
			return true, nil
		}

		return false, fmt.Errorf("error parsing existing certificate: %w", parseErr)
	}

	return false, nil
}
