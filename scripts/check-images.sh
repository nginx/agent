#!/usr/bin/env bash
set -euo pipefail
#set -x # Uncomment for debugging

usage() {
    echo "$0"
    echo
    echo "  Check and pull NGINX Agent Docker images from the official NGINX Docker registry."
    echo "    Args:"
    echo "      1. Registry URL (default: docker-registry.nginx.com/nginx)"
    echo "      2. Image Name (default: agentv3)"
    echo "      3. Search Pattern (optional): A regex pattern to filter tags before checking versions"
    echo
    echo "  Usage:"
    echo "    > $0 <registry-url> <image-name> [search-pattern]"
    echo
    echo "  Example:"
    echo "    Search for all tags for the 'agentv3' image in the NGINX Docker registry:"
    echo "      > $0 docker-registry.nginx.com/nginx agentv3"
    echo
    echo "    Search for all tags containing 'alpine' for the 'agentv3' image in the NGINX Docker registry:"
    echo "      > $0 docker-registry.nginx.com/nginx agentv3 alpine"
    exit 0
}

while getopts "h" opt; do
    case ${opt} in
        h )
            usage
            ;;
        \? )
            usage
            ;;
    esac
done

# Input parameters with defaults
REGISTRY_URL=${1:-"docker-registry.nginx.com/nginx"}
IMAGE_NAME=${2:-"agentv3"}
RE_PATTERN=${3:-""}
IMAGE_PATH="${REGISTRY_URL}/${IMAGE_NAME}"
CONTAINER_TOOL=docker
SKOPEO_IMAGE="quay.io/skopeo/stable"
SKOPEO_TAG="latest"

# Check for docker installation
if ! command -v ${CONTAINER_TOOL} &> /dev/null; then
    echo "${CONTAINER_TOOL} could not be found."
    # check podman as an alternative
    CONTAINER_TOOL=podman
    if ! command -v ${CONTAINER_TOOL} &> /dev/null; then
        echo "Neither docker nor podman could be found. Please install one of them to proceed."
        exit 1
    fi
fi

echo "Using container tool: ${CONTAINER_TOOL}"
${CONTAINER_TOOL} --version

echo "Getting skopeo tool..."
${CONTAINER_TOOL} pull ${SKOPEO_IMAGE}:${SKOPEO_TAG} || { echo "Failed to pull skopeo image"; exit 1; }

echo "Checking images in ${REGISTRY_URL}/${IMAGE_NAME}"
echo "Saving all tags to ${IMAGE_NAME}_tags.txt"
${CONTAINER_TOOL} run quay.io/skopeo/stable list-tags docker://${IMAGE_PATH} | jq -r '.Tags[]' > ${IMAGE_NAME}_tags.txt
echo $(wc -l < ${IMAGE_NAME}_tags.txt) "tags fetched."

# Filter out tags that end with four or more digits (nightly/build tags)
grep -Ev '\d{4,}$' ${IMAGE_NAME}_tags.txt | sort -u > ${IMAGE_NAME}_filteredtags.txt
echo $(wc -l < ${IMAGE_NAME}_filteredtags.txt) "tags after filtering."

# Search for tags matching the provided pattern
FOUND=($(grep -E "${RE_PATTERN}" ${IMAGE_NAME}_filteredtags.txt))
echo "tags matching '${RE_PATTERN}':" ${#FOUND[@]}
echo "${FOUND[@]}" | sed 's/ /\n/g'

for tag in "${FOUND[@]}"; do
    echo ":: ${IMAGE_PATH}:$tag"
    ${CONTAINER_TOOL} pull ${IMAGE_PATH}:$tag > /dev/null 2>&1
    echo -n ":::: "; ${CONTAINER_TOOL} run ${IMAGE_PATH}:$tag cat /etc/os-release | grep PRETTY_NAME | awk -F'=' '{print $2}' | tr -d '"' \
      || echo "No /etc/os-release found"
    echo -n ":::: "; ${CONTAINER_TOOL} run ${IMAGE_PATH}:$tag nginx -v
    echo -n ":::: "; ${CONTAINER_TOOL} run --rm ${IMAGE_PATH}:$tag nginx-agent --version | sed 's/version/version:/g' # --rm to clean up container after run
    echo
done

