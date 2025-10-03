#!/bin/bash

PROFILE=${PROFILE:-"false"}
BENCHMARKS_DIR="/agent/performance/load"

load_test() {
  echo "Running load tests..."
  pushd test/load
  go test -v -timeout 1m ./...
  cp benchmarks.json ${BENCHMARKS_DIR}
  cp -r results ${BENCHMARKS_DIR}
  popd
}

load_test_with_profile() {
  echo "Running load tests with CPU Profiling enabled..."
  pushd test/load
  go test -v -timeout 1m ./... \
    -cpuprofile metrics_load_cpu.pprof
  cp benchmarks.json ${BENCHMARKS_DIR}
  cp -r results ${BENCHMARKS_DIR}
  cp *.pprof ${BENCHMARKS_DIR}
  popd
}

## Main script execution starts here
mkdir -p ${BENCHMARKS_DIR}
echo "Running in $(pwd)"
if [[ "$PROFILE" == "true" ]]; then
  echo "CPU Profiling is enabled."
  load_test_with_profile || { echo "Load tests with cpu profiling failed"; exit 1; }
else
  load_test || { echo "Load tests failed"; exit 1; }
fi
echo "Done."
