#!/bin/sh

help() {
    echo "Usage:
$0 (ca | intermediate | end-entity) (rsa | dsa) --config <config_path> [opts...] --out <output_dir>
    opts:
        --ca-cert <cert_path>       Path to signing entity certificate
        --ca-key <key_path>         Path to signing entity key"
}

CONFIG=""
CA_CERT=""
CA_KEY=""
OUT=""

parse_args() {
    shift
    shift

    while [ "$#" -gt 0 ]; do
        case $1 in
            "--config")
                shift
                if [ -f "$1" ] && [ -r "$1" ]; then
                    CONFIG="$1"
                else
                    echo "! Config file does not exist or unreadable: $1"
                    exit 1
                fi
                ;;
            "--ca-cert")
                shift
                if [ -f "$1" ] && [ -r "$1" ]; then
                    CA_CERT="$1"
                else
                    echo "! CA cert file does not exist or unreadable: $1"
                    exit 1
                fi
                ;;
            "--ca-key")
                shift
                if [ -f "$1" ] && [ -r "$1" ]; then
                    CA_KEY="$1"
                else
                    echo "! CA key file does not exist or unreadable: $1"
                    exit 1
                fi
                ;;
            "--out")
                shift
                if [ -d "$1" ]; then
                    OUT="$1"
                else
                    if mkdir -p "$1"; then
                        OUT="$1"
                    else
                        echo "! Could not create output directory: $1"
                        exit 1
                    fi
                fi
                ;;
            *)
                help
                echo "! Unrecognised Option: $1"
                exit 1
        esac
        shift
    done

    if [ -z "$CONFIG" ]; then
        echo "! --config <config_path> not specified"
        exit 1
    fi

    if [ -z "$OUT" ]; then
        echo "! --out <output_directory> not specified"
        exit 1
    fi
}

create_self_signed_dsa() {
    if ! openssl gendsa \
        -out "$OUT/$1.key" <(openssl dsaparam 4096 ); then

        echo "! Failed to generate self signed cert. Verify $CONFIG is a valid config"
        exit 1
    fi

    if ! openssl req \
        -x509 \
        -sha512 \
        -key "$OUT/$1.key" \
        -out "$OUT/$1.crt" \
        -config "$CONFIG" \
        -extensions v3_req; then

        echo "! Failed to generate self signed cert. Verify $CONFIG is a valid config"
        exit 1
    fi
    chmod 644 "$OUT/$1.crt"
}

create_self_signed() {
    if ! openssl req \
        -newkey rsa:4096 \
        -nodes \
        -x509 \
        -sha512 \
        -keyout "$OUT/$1.key" \
        -out "$OUT/$1.crt" \
        -config "$CONFIG" \
        -extensions v3_req; then

        echo "! Failed to generate self signed cert. Verify $CONFIG is a valid config"
        exit 1
    fi
    chmod 644 "$OUT/$1.crt"
}

create_csr() {
    if ! openssl req \
        -newkey rsa:4096 \
        -nodes \
        -sha512 \
        -keyout "$OUT/$1.key" \
        -out "$OUT/$1.csr" \
        -config "$CONFIG" \
        -extensions v3_req; then

        echo "! Failed to generate csr. Verify $CONFIG is a valid config"
        exit 1
    fi
}

sign_csr() {
    if openssl x509 \
        -req \
        -CA "$2" \
        -CAkey "$3" \
        -CAcreateserial \
        -in "$OUT/$1.csr" \
        -out "$OUT/$1.crt" \
        -extfile "$CONFIG" \
        -extensions v3_req; then

        cat "$OUT/$1.crt" "$CA_CERT" > "$OUT/$1_fullchain.crt"
        chmod 644 "$OUT/$1.crt" "$OUT/$1_fullchain.crt"
    else
        echo "Failed to sign cert. Verify $2 and $3 are a valid cert and key pair"
        exit 1
    fi
}

if [ "$#" -lt 5 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    help "$0"
    exit 1
fi

if ! type openssl; then
    echo "! openssl binary not found"
    exit 1
fi

check_args() {
    if [ -z "$CA_CERT" ]; then
        echo "! --ca-cert <cert_path> not specified"
        exit 1
    fi

    if [ -z "$CA_KEY" ]; then
        echo "! --ca-key <key_path> not specified"
        exit 1
    fi
}


case $1 in
    "ca")
        parse_args "$@"
        if [ $2 == "rsa" ]; then
            create_self_signed "ca"
        elif [ $2 == "dsa" ]; then
            create_self_signed_dsa "ca"
        else
            echo "! illegal algorithm name" 
            echo "valid ones are rsa, dsa"
            exit 1
        fi
        ;;
    "intermediate")
        parse_args "$@"
        check_args
        create_csr "int"
        sign_csr "int" "$CA_CERT" "$CA_KEY"
        ;;
    "end-entity")
        parse_args "$@"
        check_args
        create_csr "ee"
        sign_csr "ee" "$CA_CERT" "$CA_KEY"
        ;;
    *)
        echo "! Invalid target: $1"
        help "$0"
        exit 1
        ;;
esac
