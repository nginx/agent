#!/bin/sh
set -e

if [ -f .srl ]; then
   rm .srl
   echo ".srl removed"
fi

if [ ! -d ../../build/certs ]; then
  mkdir -p ../../build/certs;
fi

make_ca() {
    echo "Creating Self-Signed Root CA certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -x509 \
        -sha256 \
        -keyout ../../build/certs/ca.key \
        -out ../../build/certs/ca.crt \
        -config ca.cnf \
        -extensions v3_req \
        -days 1
}

make_int() {
    echo "Creating Intermediate CA certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout ../../build/certs/ca_int.key \
        -out ../../build/certs/ca_int.csr \
        -config ca-intermediate.cnf \
        -extensions v3_req
    openssl req -in ../../build/certs/ca_int.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA ../../build/certs/ca.crt \
        -CAkey ../../build/certs/ca.key \
        -CAcreateserial \
        -in ../../build/certs/ca_int.csr \
        -out ../../build/certs/ca_int.crt \
        -extfile ca-intermediate.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile ../../build/certs/ca.crt ../../build/certs/ca_int.crt
    echo "Creating CA chain"
    cat ../../build/certs/ca_int.crt ../../build/certs/ca.crt > ../../build/certs/ca.pem
}

make_server() {
    echo "Creating nginx-manger certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout ../../build/certs/server.key \
        -out ../../build/certs/server.csr \
        -config server.cnf
    openssl req -in ../../build/certs/server.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA ../../build/certs/ca_int.crt \
        -CAkey ../../build/certs/ca_int.key \
        -CAcreateserial \
        -in ../../build/certs/server.csr \
        -out ../../build/certs/server.crt \
        -extfile server.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile ../../build/certs/ca.pem ../../build/certs/server.crt
}

make_client() {
    echo "Creating Client certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout ../../build/certs/client.key \
        -out ../../build/certs/client.csr \
        -config client.cnf
    openssl req -in ../../build/certs/client.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA ../../build/certs/ca.crt \
        -CAkey ../../build/certs/ca.key \
        -CAcreateserial \
        -in ../../build/certs/client.csr \
        -out ../../build/certs/client.crt \
        -extfile client.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile ../../build/certs/ca.pem ../../build/certs/client.crt
}

# MAIN
make_ca
make_int
make_server
make_client
