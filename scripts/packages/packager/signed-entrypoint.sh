#!/bin/bash

set -e
set -x
set -euxo pipefail

FREEBSD_DISTROS="FreeBSD:12:amd64 FreeBSD:13:amd64"

cd /nginx-agent/


mkdir -p ./build/packages/txz
mkdir -p ./build/github/packages

mkdir -p staging/usr/local/bin
mkdir -p staging/usr/local/etc/nginx-agent
mkdir -p staging/usr/local/etc/rc.d

cp nginx-agent.conf staging/usr/local/etc/nginx-agent
cp scripts/packages/nginx-agent staging/usr/local/etc/rc.d
cp scripts/packages/preinstall.sh staging/+PRE_INSTALL
cp scripts/packages/postremove.sh staging/+PRE_DEINSTALL
cp scripts/packages/postinstall.sh staging/+POST_INSTALL
cp scripts/packages/plist staging
cp build/nginx-agent staging/usr/local/bin

chmod +x staging/usr/local/etc/rc.d/nginx-agent

# Temporary fix until the follow issue is resolved https://github.com/actions/checkout/issues/1169
git config --global --add safe.directory /nginx-agent
VERSION="$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')" envsubst < scripts/packages/manifest > staging/+MANIFEST

for freebsd_abi in $FREEBSD_DISTROS; do \
    mkdir -p ./build/packages/txz/"$freebsd_abi"; \
    pkg -o ABI="$freebsd_abi" create --format txz \
        -m staging \
        -r staging \
        -p staging/plist \
        -o ./build/packages/txz/"$freebsd_abi"; \
    # create freebsd pkg repo layout
    pkg repo ./build/packages/txz/"$freebsd_abi" .key.rsa; \
    # Creating symbolic link from txz to pkg. In older versions of pkg the extension would represent the format of the file 
    # but since version 1.17.0 pkg will now always create a file with the extesion pkg no matter what the format is. 
    # See 1.17.0 release notes for more info: https://cgit.freebsd.org/ports/commit/?id=e497a16a286972bfcab908209b11ee6a13d99dc9
    cd build/packages/txz/"$freebsd_abi"; \
    ln -s nginx-agent-"$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')".pkg nginx-agent-"$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')".txz; \
    cd ../../../../; \
    cp ./build/packages/txz/"$freebsd_abi"/nginx-agent-"$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')".pkg ./build/github/packages/nginx-agent-"$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')"-"$freebsd_abi".pkg; \
done; \

rm -rf /staging
