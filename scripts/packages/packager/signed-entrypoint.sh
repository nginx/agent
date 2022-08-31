#!/bin/bash

set -e
set -x
set -euxo pipefail

FREEBSD_DISTROS="FreeBSD:12:amd64 FreeBSD:13:amd64"

cd /nginx-agent/

mkdir -p ./build/packages/txz

mkdir -p staging/usr/local/bin
mkdir -p staging/usr/local/etc/nginx-agent
mkdir -p staging/usr/local/etc/rc.d

cp nginx-agent.conf staging/usr/local/etc/nginx-agent
cp scripts/packages/nginx-agent staging/usr/local/etc/rc.d
cp scripts/packages/postremove.sh staging/+PRE_DEINSTALL
cp scripts/packages/postinstall.sh staging/+POST_INSTALL
cp scripts/packages/plist staging
cp build/nginx-agent staging/usr/local/bin

chmod +x staging/usr/local/etc/rc.d/nginx-agent

VERSION="$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')" envsubst < scripts/packages/manifest > staging/+MANIFEST

for freebsd_abi in $FREEBSD_DISTROS; do \
    mkdir -p ./build/packages/txz/$freebsd_abi; \
    pkg -o ABI=$freebsd_abi create --format txz \
        -m staging \
        -r staging \
        -p staging/plist \
        -o ./build/packages/txz/$freebsd_abi; \
    # create freebsd pkg repo layout \
    pkg repo ./build/packages/txz/$freebsd_abi .key.rsa; \
    mv ./build/packages/txz/$freebsd_abi/nginx-agent-$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v').pkg ./build/packages/txz/$freebsd_abi/nginx-agent-$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v').txz; \
done; \

rm -rf /staging
