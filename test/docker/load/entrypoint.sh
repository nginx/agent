#!/bin/bash

PROFILE=${PROFILE:-"false"}
BENCHMARKS_DIR="/agent/performance/load"

load_test() {
  echo "Running load tests..."
  pushd test/load
  go test -v -timeout 1m ./...

  cp benchmarks.json ${BENCHMARKS_DIR}
  cp -r results ${BENCHMARKS_DIR}/results
  popd
}

load_test_with_profile() {
  echo "Running load tests with CPU Profiling enabled..."
  pushd test/load
  go test -v -timeout 1m ./... \
    -cpuprofile ${BENCHMARKS_DIR}/metrics_load_cpu.pprof

  cp benchmarks.json ${BENCHMARKS_DIR}/benchmarks.json
  cp -r results ${BENCHMARKS_DIR}/results

  popd
}

## Main script execution starts here
mkdir -p ${BENCHMARKS_DIR}

echo "Running in $(pwd)"
# Run load tests
#   $PROFILE - if true, run with CPU profiling enabled
if [[ "$PROFILE" == "true" ]]; then
  echo "CPU Profiling is enabled."
  load_test_with_profile || { echo "Load tests with profiling failed"; exit 1; }
else
  load_test || { echo "Load tests failed"; exit 1; }
fi

echo "Listing contents of ${BENCHMARKS_DIR}:"
ls -la ${BENCHMARKS_DIR}

echo "Done."
