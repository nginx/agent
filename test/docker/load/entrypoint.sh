#!/bin/bash

PROFILE=${PROFILE:-"false"}
DESTINATION="/agent/performance"
BENCHMARKS_DIR="${DESTINATION}/load-tests"
mkdir -p ${BENCHMARKS_DIR}

load_test() {
  echo "Running load tests..."
  echo "Results will be saved to ${BENCHMARKS_DIR}/benchmarks.json"
  go test -v -timeout 1m ./test/load/... \
      | tee ${BENCHMARKS_DIR}/benchmarks.json || { echo "Load tests failed, exiting..."; exit 1; }
  find ${BENCHMARKS_DIR}
}

load_test_with_profile() {
  echo "Running load tests with CPU Profiling enabled..."
  mkdir -p ${BENCHMARKS_DIR}
  go test -v -timeout 1m ./test/load/... \
      -cpuprofile ${BENCHMARKS_DIR}/metrics_cpu.pprof \
      | tee ${BENCHMARKS_DIR}/benchmarks.json || { echo "Load tests failed, exiting..."; exit 1; }
  cp ${BENCHMARKS_DIR}/metrics_load_cpu.pprof ${BENCHMARKS_DIR}/profile.pgo
  find ${BENCHMARKS_DIR}
}

copy_agent_files() {
  echo "Copying agent files..."
  cp /agent/test/load/results/TestMetric10kDPS/OTLP-linux-build/nginx-agent/agent.log ${BENCHMARKS_DIR}/agent.log \
    || echo "No agent.log file found"
  cp /agent/test/load/results/TestMetric10kDPS/OTLP-linux-build/nginx-agent/opentelemetry-collector-agent.log ${BENCHMARKS_DIR}/otel.log \
    || echo "No opentelemetry-collector-agent.log file found"
}

## Main script execution starts here

mkdir -p $DESTINATION

echo "Running in $(pwd)"
# Run load tests
#   $PROFILE - if true, run with CPU profiling enabled
if [[ "$PROFILE" == "true" ]]; then
  echo "CPU Profiling is enabled."
  load_test_with_profile
else
  load_test
fi

copy_agent_files
echo "Done."
