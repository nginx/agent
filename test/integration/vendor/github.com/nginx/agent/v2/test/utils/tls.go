package utils

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc/credentials"
)

func LoadKeyPair() credentials.TransportCredentials {
	certificate, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		log.Fatalf("Load server certification failed: %v", err)
	}

	data, err := ioutil.ReadFile("certs/client/ca.crt")
	if err != nil {
		log.Fatalf("can't read ca file: %v", err)
	}

	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM(data) {
		log.Fatal("can't add ca cert")
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    capool,
	}
	return credentials.NewTLS(tlsConfig)
}
