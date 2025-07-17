#!/bin/bash
#   Usage:
#
#        Check package v3.0.0 availability for all platforms, no auth required:
#            > ./package_check.sh 3.0.0
#
#        Check pkgs and download if present, with authentication:
#            > CERT=<cert-path> KEY=<key-path> DL=1 ./package_check.sh 3.0.0
#
#        Required parameters:
#
#            version: the version of agent you wish to search for i.e 3.0.0
#
#        Optional parameters:
#
#            PKG_REPO: The root url for the repository you wish to check, defaults to packages.nginx.org
#            CERT: Path to your cert file
#            KEY: Path to your key file
#            DL: Switch to download the package if it is present, set to 1 if download required, defaults to 0
#
#   Packages are downloaded to the local directory with the path of its corresponding repo url + uri i.e
#
#            packages.nginx.org/nginx-agent/debian/pool/agent/n/nginx-agent/nginx-agent_3.0.0~bullseye_arm64.deb
#


RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

CURL_OPTS=""
if [[ ! -z ${CERT} ]] && [[ ! -z ${KEY} ]]; then
  CURL_OPTS="-E ${CERT} --key ${KEY}"
fi

if [[ -z ${PKG_REPO} ]]; then
  echo "defaulting to packages.nginx.com"
  PKG_REPO="packages.nginx.org"
fi

PKG_NAME="nginx-agent"
VERSION="${1}"
if [[ -z $VERSION ]]; then
  echo "no version provided"
  exit 1
fi

PKG_DIR="${PKG_REPO}/${PKG_NAME}"
PKG_REPO_URL="https://${PKG_DIR}"

APK=(
  alpine/v3.22/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.22/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.21/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.21/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.20/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.20/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.19/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.19/main/x86_64/nginx-agent-$VERSION.apk
)
UBUNTU=(
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~jammy_amd64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~noble_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~plucky_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~jammy_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~noble_amd64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~plucky_amd64.deb
)
DEBIAN=(
  debian/pool/agent/n/nginx-agent/nginx-agent_$VERSION~bullseye_arm64.deb
  debian/pool/agent/n/nginx-agent/nginx-agent_$VERSION~bookworm_amd64.deb
  debian/pool/agent/n/nginx-agent/nginx-agent_$VERSION~bookworm_arm64.deb
  debian/pool/agent/n/nginx-agent/nginx-agent_$VERSION~bullseye_amd64.deb
)
AMZN=(
  amzn/2023/aarch64/RPMS/nginx-agent-$VERSION.amzn2023.ngx.aarch64.rpm
  amzn/2023/x86_64/RPMS/nginx-agent-$VERSION.amzn2023.ngx.x86_64.rpm

  amzn2/2/aarch64/RPMS/nginx-agent-$VERSION.amzn2.ngx.aarch64.rpm
  amzn2/2/x86_64/RPMS/nginx-agent-$VERSION.amzn2.ngx.x86_64.rpm
)
SUSE=(
  sles/15/x86_64/RPMS/nginx-agent-$VERSION.sles15.ngx.x86_64.rpm
)
CENTOS=(
  centos/10/aarch64/RPMS/nginx-agent-$VERSION.el10.ngx.aarch64.rpm
  centos/10/x86_64/RPMS/nginx-agent-$VERSION.el10.ngx.x86_64.rpm
  centos/9/aarch64/RPMS/nginx-agent-$VERSION.el9.ngx.aarch64.rpm
  centos/9/x86_64/RPMS/nginx-agent-$VERSION.el9.ngx.x86_64.rpm
  centos/8/aarch64/RPMS/nginx-agent-$VERSION.el8.ngx.aarch64.rpm
  centos/8/x86_64/RPMS/nginx-agent-$VERSION.el8.ngx.x86_64.rpm
)

uris=(
  ${DEBIAN[@]}
  ${UBUNTU[@]}
  ${CENTOS[@]}
  ${APK[@]}
  ${AMZN[@]}
  ${SUSE[@]}
)

## Check and download if nginx-agent packages are present in the repository
check_pkgs () {
  for pkg in ${uris[@]}; do
    echo -n "CHECK: ${PKG_REPO_URL}/${pkg} -> "
    local ret=$(curl -I -s ${CURL_OPTS} "https://${PKG_DIR}/${pkg}" | head -n1 | awk '{ print $2 }')
    if [[ ${ret} != 200 ]]; then
      echo -e "${RED}${ret}${NC}"
      continue
    fi
    echo -e "${GREEN}${ret}${NC}"

    if [[ ${DL} == 1 ]]; then
      dl_pkg "${PKG_REPO_URL}/${pkg}"
    fi

  done
}

## Download a package
dl_pkg () {
    local url=${1}
    echo -n "GET: ${url}... "
    mkdir -p "${PKG_DIR}/$(dirname ${pkg})"
    local ret=$(curl -s ${CURL_OPTS} "${url}" --output "${PKG_DIR}/${pkg}")
    if [[ $? != 0 ]]; then
      echo -e "${RED}Download failed!${NC}"
      return
    fi
    echo -e "${GREEN}Done${NC}"
    echo "SAVED: ${PKG_DIR}/${pkg}"
}

## Check for the presence of an nginx-agent version matching $VERSION
check_repo() {
  echo -n "Checking package repository ${PKG_REPO_URL}... "
  curl -s -I ${CURL_OPTS} "${PKG_REPO_URL}/index.xml" > /dev/null
  if [[ $? != 0 ]]; then
    echo -e "${RED}index.xml not found in ${PKG_REPO_URL} repository${NC}"
    exit 1
  else
    echo -e "${GREEN}Found!${NC}"
  fi

  mkdir -p ${PKG_DIR}
  curl -s ${CURL_OPTS} "${PKG_REPO_URL}/index.xml" --output "${PKG_DIR}/index.xml" || exit 1

  echo -n "Checking for nginx-agent version ${VERSION}... "
  grep -qnF "ver=\"${VERSION}\"" "${PKG_DIR}/index.xml"
  if [[ $? != 0 ]]; then
    echo -e "${RED}not found${NC}"
    exit 1
  else
    echo -e "${GREEN}Found!${NC}"
  fi
}

check_repo
check_pkgs
