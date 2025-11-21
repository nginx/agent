#!/bin/bash
#
#   package_check.sh
#
#   Check for the presence of nginx-agent packages in a given package repository,
#   and download them if required. Defaults to packages.nginx.org if no repository is specified.
#
#   If DL=1 and files are downloaded, it also prepares the packages for upload to Azure and GitHub Releases by
#   modifying paths to match the expected format for each platform.
#
#   Usage:
#
#        Check package v3.0.0 availability for all platforms, no auth required:
#            > ./package_check.sh 3.0.0
#
#        Check pkgs and download if present, with authentication:
#            > DL=1 CERT=<cert-path> KEY=<key-path> ./package_check.sh 3.5.0
#
#        Required parameters:
#
#            version: the version of agent you wish to search for i.e 3.5.0
#
#        Optional env variables:
#
#            PKG_REPO: The root url for the repository you wish to check, defaults to packages.nginx.org
#            CERT: Path to your cert file
#            KEY: Path to your key file
#            DL: Switch to download the package if it is present, set to 1 if download required, defaults to 0
#
#   Packages are downloaded to the local directory with the path of its corresponding repo url + uri, i.e
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
  echo "defaulting to packages.nginx.org"
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
DL=${DL:-0}

majorVersion=$(echo ${VERSION} | cut -d. -f1)

# Define package URIs to check for each platform

APK=()
ALPINE_VERSIONS=("3.22" "3.21" "3.20" "3.19")
ALPINE_ARCH=("x86_64" "aarch64")
for alpine_version in "${ALPINE_VERSIONS[@]}"; do
    for arch in ${ALPINE_ARCH[@]}; do
      APK+=("alpine/v${alpine_version}/main/${arch}/nginx-agent-${VERSION}.apk")
    done
done

UBUNTU=()
UBUNTU_VERSIONS=("jammy" "noble" "plucky")
DEB_ARCH=("amd64" "arm64")
for ubuntu_version in "${UBUNTU_VERSIONS[@]}"; do
    for arch in ${DEB_ARCH[@]}; do
      UBUNTU+=("ubuntu/pool/agent/n/nginx-agent/nginx-agent_${VERSION}~${ubuntu_version}_${arch}.deb")
    done
done

DEBIAN=()
DEBIAN_VERSIONS=("bullseye" "bookworm" "trixie")
for deb_version in "${DEBIAN_VERSIONS[@]}"; do
    for arch in ${DEB_ARCH[@]}; do
      DEBIAN+=("debian/pool/agent/n/nginx-agent/nginx-agent_${VERSION}~${deb_version}_${arch}.deb")
    done
done

CENTOS=()
CENTOS_VERSIONS=("10" "9" "8")
RPM_ARCH=("aarch64" "x86_64")
for centos_version in "${CENTOS_VERSIONS[@]}"; do
    for arch in ${RPM_ARCH[@]}; do
      CENTOS+=("centos/${centos_version}/${arch}/RPMS/nginx-agent-${VERSION}.el${centos_version}.ngx.${arch}.rpm")
    done
done

AMZN=()
for arch in ${RPM_ARCH[@]}; do
    AMZN+=("amzn/2023/${arch}/RPMS/nginx-agent-$VERSION.amzn2023.ngx.${arch}.rpm")
    AMZN+=("amzn2/2/${arch}/RPMS/nginx-agent-$VERSION.amzn2.ngx.${arch}.rpm")
done

SUSE=(
  sles/15/x86_64/RPMS/nginx-agent-$VERSION.sles15.ngx.x86_64.rpm
)

# Aggregate all URIs to fetch
uris=(
  ${DEBIAN[@]}
  ${UBUNTU[@]}
  ${CENTOS[@]}
  ${APK[@]}
  ${AMZN[@]}
  ${SUSE[@]}
)

if [[ ${majorVersion} == 2 ]]; then
  # v2.x supports FreeBSD packages
  FREEBSD=(
    freebsd/FreeBSD:12:amd64/latest/nginx-agent-$VERSION.pkg
    freebsd/FreeBSD:13:amd64/latest/nginx-agent-$VERSION.pkg
  )
  uris+=(${FREEBSD[@]})
fi

# Functions

## Check and download if nginx-agent packages are present in the repository
check_pkgs() {
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
dl_pkg() {
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

# Prepare packages for upload to Azure
prep_deb() {
  echo "Preparing deb packages..."
  mkdir -p "${PKG_DIR}/azure/deb"
  for i in $(find "${PKG_DIR}" | grep -e "nginx-agent[_-]${VERSION}.*\.deb" | grep -v "azure"); do
    az_dest="${PKG_DIR}/azure/deb/$(basename "$i")"
    # Azure path
    # Should be 'nginx-agent_3.5.0~bullseye_arm64.deb'
    echo "Copying ${i} to ${az_dest}"
    cp "${i}" "${az_dest}"
    # GitHub release asset path
    # Should be 'nginx-agent-3.5.0.bullseye_arm64.deb'
    gh_dest=$(echo ${i} | sed -E "s/_/-/" | sed -E "s/~/./")
    echo "Moving ${i} to ${gh_dest}"
    mv "${i}" "${gh_dest}"
  done
}

prep_apk() {
  echo "Preparing apk packages..."
  mkdir -p "${PKG_DIR}/azure/apk"
  for i in $(find "${PKG_DIR}/alpine" | grep -e "nginx-agent[_-]${VERSION}.apk"); do
    ver=$(echo "$i" | grep -o -e "v[0-9]*\.[0-9]*")
    arch=$(echo "$i" | grep -o -F -e "x86_64" -e "aarch64")
    dest="$(dirname "$i")/nginx-agent-${VERSION}-$ver-$arch.apk"
    az_dest="${PKG_DIR}/azure/apk/$ver/$arch/nginx-agent-${VERSION}.apk"

    # Azure path
    mkdir -p "$(dirname "$az_dest")"
    echo "Copying ${i} to ${az_dest}"
    cp "${i}" "${az_dest}"

    # GitHub release asset path
    echo "Moving ${i} to ${dest}"
    mv "${i}" "${dest}"
  done
}

prep_rpm() {
  echo "Preparing rpm packages..."
  mkdir -p "${PKG_DIR}/azure/rpm"
  for i in $(find "${PKG_DIR}" | grep -e "nginx-agent-${VERSION}.*.rpm"); do
    az_dest="${PKG_DIR}/azure/rpm/$(basename "$i")"
    # Azure path
    echo "Copying ${i} to ${az_dest}"
    mkdir -p "$(dirname "$az_dest")"
    cp "${i}" "${az_dest}"
    # No path changes needed for GitHub release
  done
}

prep_txz() {
  echo "Preparing txz packages..."
  mkdir -p "${PKG_DIR}/azure/txz"
  for i in $(find "${PKG_DIR}" | grep -e "nginx-agent[_-]${VERSION}.pkg"); do
    bsd=$(echo "$i" | grep -e "FreeBSD:[0-9]*")
    ver=$(echo "$bsd" | cut -d':' -f2)
    arch=$(echo "$i" | grep -o -F -e "amd64" -e "arm64")
    dest="$(dirname "$i")/nginx-agent-${VERSION}-FreeBSD.$ver.$arch.pkg"
    az_dest="${PKG_DIR}/azure/txz/FreeBSD:$ver:$arch/nginx-agent-${VERSION}.pkg"
    echo "Copying ${i} to ${az_dest/latest\//}"
    mkdir -p "$(dirname "$az_dest")"
    cp "${i}" "${az_dest}"
  done
}

prepare_packages() {
  echo "Preparing packages for upload..."
  prep_deb
  prep_apk
  prep_rpm
  if [[ ${majorVersion} == 2 ]]; then
    prep_txz
  fi

  echo
  echo "Prepared packages for Azure:"
  find "${PKG_DIR}/azure" -type f | grep "${VERSION}" | sed "s|${PKG_DIR}/azure/||" | sort

  echo
  echo "Prepared packages for GitHub Release v${VERSION}:"
  find "${PKG_DIR}" -type f | grep "${VERSION}" | grep -v "/azure/" | awk -F/ '{print $NF}' | sort
}

create_tarball() {
  echo "Creating tarball... "
  tar -czvf "${PKG_DIR}/nginx-agent.tar.gz" -C ${PKG_DIR}/azure .
  if [[ $? != 0 ]]; then
    echo -e "${RED}Failed to create tarball!${NC}"
    return
  fi
  echo "SAVED: ${PKG_DIR}/nginx-agent.tar.gz"
}

# Main

check_repo
check_pkgs

# Prepare packages for upload
if [[ ${DL} == 1 ]]; then
  prepare_packages
  create_tarball
fi
