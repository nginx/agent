package sdk

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
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
	fileContents, err := ioutil.ReadFile(certPath)
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
