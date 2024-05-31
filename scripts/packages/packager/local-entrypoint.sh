#!/bin/bash

set -e
set -x
set -euxo pipefail

VERSION_TAG=$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')
COMMIT_SHA=$(git rev-parse --short HEAD)
export VERSION=${VERSION:-"${VERSION_TAG}-SNAPSHOT-${COMMIT_SHA}"}

case "$(uname -m)" in
    amd64|x86_64)  ABIARCH=amd64 ;;
    arm64|aarch64) ABIARCH=aarch64 ;;
esac

cd /nginx-agent/

mkdir -p /staging/usr/local/bin
mkdir -p /staging/usr/local/etc/nginx-agent
mkdir -p /staging/usr/local/etc/rc.d

cp nginx-agent.conf /staging/usr/local/etc/nginx-agent
cp scripts/packages/nginx-agent /staging/usr/local/etc/rc.d
cp scripts/packages/preinstall.sh /staging/+PRE_INSTALL
cp scripts/packages/postremove.sh /staging/+POST_DEINSTALL
cp scripts/packages/postinstall.sh /staging/+POST_INSTALL
cp scripts/packages/plist /staging
cp build/nginx-agent /staging/usr/local/bin

chmod +x /staging/usr/local/etc/rc.d/nginx-agent

# Temporary fix until the follow issue is resolved https://github.com/actions/checkout/issues/1169
git config --global --add safe.directory /nginx-agent
envsubst < scripts/packages/manifest > /staging/+MANIFEST

mkdir -p ./build

pkg -o ABI="FreeBSD:13:${ABIARCH}" create --format txz \
    -m /staging \
    -r /staging \
    -p /staging/plist \
    -o ./build

# Creating symbolic link from txz to pkg. In older versions of pkg the extension would represent the format of the file
# but since version 1.17.0 pkg will now always create a file with the extesion pkg no matter what the format is.
# See 1.17.0 release notes for more info: https://cgit.freebsd.org/ports/commit/?id=e497a16a286972bfcab908209b11ee6a13d99dc9
cd build
ln -s "nginx-agent-${VERSION}.pkg" "nginx-agent-${VERSION}.txz"
cd ../

rm -rf /staging
