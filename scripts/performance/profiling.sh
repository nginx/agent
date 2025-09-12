#!/usr/bin/env bash

# This script runs Go tests with CPU profiling enabled for all test packages in the test/integration directory.
# It saves the CPU profiles in the profiles directory with the format <package-name>_<test_type>.pprof
# Usage: ./scripts/performance/profiling.sh
#
# Variables:
#   - TEST_ENV: The test environment (default: "local")
#   - CONTAINER_OS_TYPE: The container OS type (default: "linux")
#   - BUILD_TARGET: The build target (default: "agent")
#   - PACKAGES_REPO: The packages repository (default: "packages.nginx.org")
#   - PACKAGE_NAME: The package name (default: "nginx-agent
#   - BASE_IMAGE: The base image (default: "ubuntu")
#   - OS_VERSION: The OS version (default: "20.04")
#   - OS_RELEASE: The OS release (default: "focal")
#   - DOCKERFILE_PATH: The Dockerfile path (default: "Dockerfile")
#   - IMAGE_PATH: The image path (default: "nginxinc/nginx-agent")
#   - TAG: The image tag (default: "latest")
#   - CONTAINER_NGINX_IMAGE_REGISTRY: The container registry (default: "docker.io")
# Example:
#   TEST_ENV=ci CONTAINER_OS_TYPE=linux BUILD_TARGET=agent PACKAGES_REPO=nginxinc PACKAGE_NAME=nginx-agent BASE_IMAGE=ubuntu OS_VERSION=20.04 OS_RELEASE=focal DOCKERFILE_PATH=Dockerfile IMAGE_PATH=nginxinc/nginx-agent TAG=latest CONTAINER_NGINX_IMAGE_REGISTRY=docker.io ./scripts/performance/profiling.sh

## Print all variables in the environment
#echo "TEST_ENV=$TEST_ENV"
#echo "CONTAINER_OS_TYPE=$CONTAINER_OS_TYPE"
#echo "BUILD_TARGET=$BUILD_TARGET"
#echo "PACKAGES_REPO=$PACKAGES_REPO"
#echo "PACKAGE_NAME=$PACKAGE_NAME"
#echo "BASE_IMAGE=$BASE_IMAGE"
#echo "OS_VERSION=$OS_VERSION"
#echo "OS_RELEASE=$OS_RELEASE"
#echo "DOCKERFILE_PATH=$DOCKERFILE_PATH"
#echo "IMAGE_PATH=$IMAGE_PATH"
#echo "TAG=$TAG"
#echo "CONTAINER_NGINX_IMAGE_REGISTRY=$CONTAINER_NGINX_IMAGE_REGISTRY"

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
      -count 10 -timeout 1m \
      -cpuprofile "${PROFILES_DIR}/$(basename $pkg)_watcher_cpu.pprof" \
      "./${pkg}" || { echo "Tests failed in package: ${pkg}, but continuing..."; continue; }
    echo "Profile saved to: ${PROFILES_DIR}/$(basename $pkg)_watcher_cpu.pprof"
done

## Run integration tests with CPU profiling for each package
echo "Starting integration tests cpu profiling tests..."
packages=$(find test/integration -type f -name '*_test.go' -exec dirname {} \; | sort -u)
echo "Found packages:"
echo "$packages"
for pkg in $packages; do
    echo "Running tests in package: ${pkg}"
    go test -v \
      -count 3 -timeout 1m \
      -cpuprofile "profiles/$(basename $pkg)_integration_cpu.pprof" \
      "./${pkg}" || { echo "Tests failed in package: ${pkg}, but continuing..."; continue; }
    mv $(basename $pkg).test profiles/$(basename $pkg)_integration_cpu.test
    echo "Profile saved to: profiles/${profile_name}"
done

## Merge all CPU profiles
files=$(ls ${PROFILES_DIR}/*.pprof)
echo "Merging CPU profiles: $files"
go tool pprof -proto -output=${PROFILES_DIR}/merged.pgo $files
echo "Merged CPU profile saved to: default.pgo"

## Cleanup
rm *.pprof
