#!/usr/bin/env bash
# See https://stackoverflow.com/questions/34612991/openssl-key-generation-on-os-x-failing
# On Mac OS X use openssl to run this script
set -e

scripts_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
echo "$scripts_dir"

build_dir="$scripts_dir/../../build/certs"
echo "$build_dir"

if [ ! -d "$build_dir" ]; then
  echo "creating certs directory"
  mkdir -p "$build_dir";
fi

ca() {
    echo "Creating Self-Signed Root CA certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -x509 \
        -sha256 \
        -keyout "$build_dir"/ca.key \
        -out "$build_dir"/ca.crt \
        -config "$scripts_dir"/ca.cnf \
        -extensions v3_req \
        -days 1
}

intermediate() {
    echo "Creating Intermediate CA certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout "$build_dir"/ca_int.key \
        -out "$build_dir"/ca_int.csr \
        -config "$scripts_dir"/ca-intermediate.cnf \
        -extensions v3_req
    openssl req -in "$build_dir"/ca_int.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA "$build_dir"/ca.crt \
        -CAkey "$build_dir"/ca.key \
        -CAcreateserial \
        -in "$build_dir"/ca_int.csr \
        -out "$build_dir"/ca_int.crt \
        -extfile "$scripts_dir"/ca-intermediate.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile "$build_dir"/ca.crt "$build_dir"/ca_int.crt
    echo "Creating CA chain"
    cat "$build_dir"/ca_int.crt "$build_dir"/ca.crt > "$build_dir"/ca.pem
}

server() {
    echo "Creating nginx-manger certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout "$build_dir"/server.key \
        -out "$build_dir"/server.csr \
        -config "$scripts_dir"/server.cnf
    openssl req -in "$build_dir"/server.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA "$build_dir"/ca_int.crt \
        -CAkey "$build_dir"/ca_int.key \
        -CAcreateserial \
        -in "$build_dir"/server.csr \
        -out "$build_dir"/server.crt \
        -extfile "$scripts_dir"/server.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile "$build_dir"/ca.pem "$build_dir"/server.crt
}

client() {
    echo "Creating Client certificate and key"
    openssl req \
        -new -newkey rsa:4096 \
        -nodes \
        -keyout "$build_dir"/client.key \
        -out "$build_dir"/client.csr \
        -config "$scripts_dir"/client.cnf
    openssl req -in "$build_dir"/client.csr -noout -verify
    openssl x509 \
        -req \
        -sha256 \
        -CA "$build_dir"/ca.crt \
        -CAkey "$build_dir"/ca.key \
        -CAserial "$build_dir"/ca_int.srl \
        -in "$build_dir"/client.csr \
        -out "$build_dir"/client.crt \
        -extfile "$scripts_dir"/client.cnf \
        -extensions v3_req \
        -days 1
    openssl verify -CAfile "$build_dir"/ca.pem "$build_dir"/client.crt
}

# MAIN
cd "$scripts_dir"
ca
intermediate
server
client
