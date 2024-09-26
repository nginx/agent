#!/bin/bash

# run load tests 
go test -v -timeout 30s -cpuprofile load_cpu.pprof

# copy profile to build for use in generate-pgo-profile
cp load_cpu.pprof /agent/build/load_cpu.pprof
cp benchmarks.json /agent/build/benchmarks.json

# cat logs & results for debugging
cat /var/log/nginx-agent/agent.log
cat /var/log/nginx-agent/opentelemetry-collector-agent.log
cat /agent/build/benchmarks.json
