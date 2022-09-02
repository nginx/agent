#!/bin/sh


help() {
        echo "Usage:
$0 (ca | intermediate | end-entity) --cn <common_name> [opts...] --out <output_dir>
    opts:
        [--san <san>]...            Subject Alternative Name
        --key-size <size>           Key size in bits (Default: 4096)
        --days <amt>                Days of validity (Default: 365)

        --country <country>         Two letter country code
        --state <state>             State or Province name
        --locality <locality>       Locality name
        --org <org>                 Organization name
        --orgunit <orgunit>         Organizational unit name"
}

OUT=""
CN=""

KEY_SIZE=""
DAYS=""
SAN=""
COUNTRY=""
STATE=""
LOCALITY=""
ORG=""
ORGUNIT=""

parse_args() {
    shift

    while [ "$#" -gt 0 ]; do
        case $1 in
            "--cn")
                shift
                CN="$1"
                ;;
            "--country")
                shift
                COUNTRY="$1"
                ;;
            "--state")
                shift
                STATE="$1"
                ;;
            "--locality")
                shift
                LOCALITY="$1"
                ;;
            "--org")
                shift
                ORG="$1"
                ;;
            "--orgunit")
                shift
                ORGUNIT="$1"
                ;;
            "--san")
                shift
                SAN="$SAN$1\n"
                ;;
            "--key-size")
                shift
                if [ "$1" -gt 0 ]; then
                    KEY_SIZE="$1"
                else
                    echo "! Invalid Keysize: $1"
                    exit 1
                fi
                ;;
            "--days")
                shift
                if [ "$1" -gt 0 ]; then
                    DAYS="$1"
                else
                    echo "! Invalid Days number: $1"
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

    if [ -z "$CN" ]; then
        echo "! --cn <Common Name> not specified"
        exit 1
    elif [ -z "$OUT" ]; then
        echo "! --out <Output Directory> not specified"
        exit 1
    fi
}

if [ "$#" -lt 5 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    help "$0"
    exit 1
fi

ca() {
    CONF="[ req ]
default_bits        = ${KEY_SIZE:-"4096"}
default_days        = ${DAYS:-"365"}
distinguished_name  = req_distinguished_name
prompt              = no
default_md          = sha512
req_extensions      = v3_req
policy              = policy

[ policy ]
countryName             = optional
stateOrProvinceName     = optional
localityName            = optional
organizationName        = optional
organizationalUnitName  = optional
commonName              = supplied

[ req_distinguished_name ]
commonName = ${CN}
${COUNTRY:+"countryName = ${COUNTRY}"}
${STATE:+"stateOrProvinceName = ${STATE}"}
${LOCALITY:+"localityName = ${LOCALITY}"}
${ORG:+"organizationName = ${ORG}"}
${ORGUNIT:+"organizationalUnitName = ${ORGUNIT}"}

[ v3_req ]
basicConstraints = critical, CA:true
keyUsage = critical, keyCertSign, cRLSign
subjectKeyIdentifier = hash
${SAN:+"subjectAltName = @alt_names

[ alt_names ]
$SAN"}"

    echo "$CONF" > "$OUT/$1.cnf"
}

end_entity() {
        CONF="[ req ]
default_bits        = ${KEY_SIZE:-"4096"}
default_days        = ${DAYS:-"365"}
distinguished_name  = req_distinguished_name
prompt              = no
default_md          = sha512
req_extensions      = v3_req
policy              = policy

[ policy ]
countryName             = optional
stateOrProvinceName     = optional
localityName            = optional
organizationName        = optional
organizationalUnitName  = optional
commonName              = supplied

[ req_distinguished_name ]
commonName = ${CN}
${COUNTRY:+"countryName = ${COUNTRY}"}
${STATE:+"stateOrProvinceName = ${STATE}"}
${LOCALITY:+"localityName = ${LOCALITY}"}
${ORG:+"organizationName = ${ORG}"}
${ORGUNIT:+"organizationalUnitName = ${ORGUNIT}"}

[ v3_req ]
${SAN:+"subjectAltName = @alt_names

[ alt_names ]
$SAN"}"

    echo "$CONF" > "$OUT/ee.cnf"
}

case $1 in
    "ca")
        parse_args "$@"
        ca "ca"
        ;;
    "intermediate")
        parse_args "$@"
        ca "int"
        ;;
    "end-entity")
        parse_args "$@"
        end_entity
        ;;
    *)
        echo "! Invalid target: $1"
        help "$0"
        exit 1
        ;;
esac
