#!/bin/bash

set -e
set -x
set -euxo pipefail

cd /nginx-agent/

mkdir -p /staging/usr/local/bin
mkdir -p /staging/usr/local/etc/nginx-agent
mkdir -p /staging/usr/local/etc/rc.d

cp nginx-agent.conf /staging/usr/local/etc/nginx-agent
cp scripts/packages/nginx-agent /staging/usr/local/etc/rc.d
cp scripts/packages/postremove.sh /staging/+PRE_DEINSTALL
cp scripts/packages/postinstall.sh /staging/+POST_INSTALL
cp scripts/packages/plist /staging
cp build/nginx-agent /staging/usr/local/bin

chmod +x /staging/usr/local/etc/rc.d/nginx-agent

VERSION="$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')-SNAPSHOT" envsubst < scripts/packages/manifest > /staging/+MANIFEST

pkg -o ABI="FreeBSD:13:amd64" create --format txz  \
    -m /staging \
    -r /staging \
    -p /staging/plist \
    -o ./build/; \

mv ./build/nginx-agent-$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')-SNAPSHOT.pkg ./build/nginx-agent-$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')-SNAPSHOT.txz 

rm -rf /staging
