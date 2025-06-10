#!/bin/bash
# Script to help check for the presence of nginx-agent packages

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

if [[ -z ${CERT} ]] || [[ -z ${KEY} ]]; then
  echo "Missing required credential"
  exit 1
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

# parsing xml in bash
#read_dom () {
#    local IFS=\>
#    read -d \< ENTITY CONTENT
#    local ret=$?
#    TAG_NAME=${ENTITY%% *}
#    ATTRIBUTES=${ENTITY#* }
#    return $ret
#}
#
#while read_dom; do
#  if
#done < ${PKG_REPO}/index.xml

APK=(
  alpine/v3.21/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.21/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.20/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.20/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.18/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.18/main/x86_64/nginx-agent-$VERSION.apk
  alpine/v3.19/main/aarch64/nginx-agent-$VERSION.apk
  alpine/v3.19/main/x86_64/nginx-agent-$VERSION.apk
)
UBUNTU=(
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~focal_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~jammy_amd64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~noble_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~jammy_arm64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~noble_amd64.deb
  ubuntu/pool/agent/n/nginx-agent/nginx-agent_$VERSION~focal_amd64.deb
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
  sles/12/x86_64/RPMS/nginx-agent-$VERSION.sles12.ngx.x86_64.rpm
)
CENTOS=(
  centos/9/aarch64/RPMS/nginx-agent-$VERSION.el9.ngx.aarch64.rpm
  centos/9/x86_64/RPMS/nginx-agent-$VERSION.el9.ngx.x86_64.rpm
  centos/8/aarch64/RPMS/nginx-agent-$VERSION.el8.ngx.aarch64.rpm
  centos/8/x86_64/RPMS/nginx-agent-$VERSION.el8.ngx.x86_64.rpm
)

resources=(
  ${DEBIAN[@]}
  ${UBUNTU[@]}
  ${CENTOS[@]}
  ${APK[@]}
  ${AMZN[@]}
  ${SUSE[@]}
)

## Check if nginx-agent packages are present and report missing or outdated files
check_pkgs () {
  for pkg in ${resources[@]}; do
    echo -n "CHECK: ${PKG_REPO_URL}/${pkg} -> "
    local ret=$(curl -I -s -E "${CERT}" --key "${KEY}" "https://${PKG_DIR}/${pkg}" | head -n1 | awk '{ print $2 }')
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

dl_pkg () {
    local url=${1}
    echo -n "GET: ${url}"
    mkdir -p "${PKG_DIR}/$(dirname ${pkg})"
    curl -s -E "${CERT}" --key "${KEY}" "https://${PKG_DIR}/${pkg}" --output "${PKG_DIR}/${pkg}" && echo -e " ${GREEN}Done${NC}" || echo -e " ${RED}Download failed!${NC}"
}

check_repo() {
  echo -n "Checking package repository ${PKG_REPO_URL}... "
  curl -s -I -E "${CERT}" --key "${KEY}" "${PKG_REPO_URL}/index.xml" > /dev/null
  if [[ $? != 0 ]]; then
    echo -e "${RED}*** Error: index.xml not found in $PKG_REPO repository ***${NC}"
    exit 1
  else
    echo -e "${GREEN}Found!${NC}"
  fi

  mkdir -p ${PKG_DIR}
  curl -s -E "${CERT}" --key "${KEY}" "${PKG_REPO_URL}/index.xml" --output "${PKG_DIR}/index.xml" || exit 2

  echo -n "Checking for nginx-agent version ${VERSION}... "
  grep -qnF "ver=\"${VERSION}\"" "${PKG_DIR}/index.xml"
  if [[ $? != 0 ]]; then
    echo -e "${RED}*** Error: ${PKG_NAME} v${VERSION} not found in ${PKG_REPO} ***${NC}"
    exit 1
  else
    echo -e "${GREEN}Found!${NC}"
  fi
}

check_repo
check_pkgs