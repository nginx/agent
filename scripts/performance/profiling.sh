#!/usr/bin/env bash

# This script runs Go tests with CPU profiling enabled for all test packages found under the directories:
#   - internal/watcher
#   - test/integration
# It saves the CPU profiles in the $PROFILES_DIR directory with the format <package-name>_<test_type>.pprof
#   e.g. <package-name>_watcher_cpu.pprof or <package-name>_integration_cpu.pprof

# The variables below can be set to customize the environment for the integration tests:
# Example using variables defined in our Makefile:
#   TEST_ENV=ci CONTAINER_OS_TYPE=linux BUILD_TARGET=agent PACKAGES_REPO=nginxinc \
#     PACKAGE_NAME=nginx-agent BASE_IMAGE=ubuntu OS_VERSION=20.04 OS_RELEASE=focal \
#     DOCKERFILE_PATH=Dockerfile IMAGE_PATH=nginxinc/nginx-agent TAG=latest CONTAINER_NGINX_IMAGE_REGISTRY=docker.io \
#     ./scripts/performance/profiling.sh

set -e
set -o pipefail

PROFILES_DIR="build/test/profiles"
mkdir -p ${PROFILES_DIR}

# Run watcher tests with CPU profiling for each package
echo "Starting watcher tests with cpu profiling..."
packages=$(find internal/watcher -type f -name '*_test.go' -exec dirname {} \; | sort -u)
echo "Found packages:"
echo "$packages"
for pkg in $packages; do
    echo "Running tests in package: ${pkg}"
    go test \
      -count 10 -timeout 3m \
      -cpuprofile "${PROFILES_DIR}/$(basename $pkg)_watcher_cpu.pprof" \
      "./${pkg}" || { echo "Tests failed in package: ${pkg}, but continuing..."; continue; }
    echo "Profile saved to: ${PROFILES_DIR}/$(basename $pkg)_watcher_cpu.pprof"
done

### Run integration tests with CPU profiling for each package
#echo "Starting integration tests cpu profiling tests..."
#packages=$(find test/integration -type f -name '*_test.go' -exec dirname {} \; | sort -u)
#echo "Found packages:"
#echo "$packages"
#for pkg in $packages; do
#    echo "Running tests in package: ${pkg}"
#    go test \
#      -count 3 -timeout 3m \
#      -cpuprofile "${PROFILES_DIR}/$(basename $pkg)_integration_cpu.pprof" \
#      "./${pkg}" || { echo "Tests failed in package: ${pkg}, but continuing..."; continue; }
#    echo "Profile saved to: ${PROFILES_DIR}/$(basename $pkg)_integration_cpu.pprof"
#done

## Merge all CPU profiles
files=$(ls ${PROFILES_DIR}/*.pprof)
echo "Merging CPU profiles: $files"
go tool pprof -proto -output=${PROFILES_DIR}/merged.pgo $files
echo "Merged CPU profile saved to: ${PROFILES_DIR}/merged.pgo"
