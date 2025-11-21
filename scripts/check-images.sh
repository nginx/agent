#!/usr/bin/env bash

set -euo pipefail
#set -x

REGISTRY_URL="docker-registry.nginx.com/nginx"
IMAGE_NAME=${1:-""}

search=${2:-""}

usage() {
    echo "Usage: $0 <image-name> [search-pattern]"
    echo "Example: $0 agentv3 alpine"
    exit 1
}

if [[ -z "${IMAGE_NAME}" ]]; then
    usage
fi

echo "Checking images in ${REGISTRY_URL}/${IMAGE_NAME}"

# Fetch all tags from the remote registry
skopeo list-tags docker://${REGISTRY_URL}/${IMAGE_NAME} | jq -r '.Tags[]' > all_tags.txt
echo $(wc -l < all_tags.txt) "tags fetched."

# Filter out tags that end with three or more digits (nightly/build tags)
grep -Ev '\d{3,}$' all_tags.txt | sort -u > filtered_tags.txt
echo $(wc -l < filtered_tags.txt) "tags after filtering."

FOUND=($(grep -E "${search}" filtered_tags.txt | sort)) || { echo "No tags found matching '${search}'"; exit 1; }
echo "tags matching '${search}':" ${#FOUND[@]}

for tag in "${FOUND[@]}"; do
    echo ":: ${REGISTRY_URL}/${IMAGE_NAME}:$tag"
    podman pull ${REGISTRY_URL}/${IMAGE_NAME}:$tag > /dev/null 2>&1
    podman run ${REGISTRY_URL}/${IMAGE_NAME}:$tag nginx -v
    podman run ${REGISTRY_URL}/${IMAGE_NAME}:$tag nginx-agent --version
    podman rm -f $(podman ps -a -q) > /dev/null 2>&1 || true
done

