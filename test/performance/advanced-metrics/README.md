# This is the benchmarking code for everything related to Advanced Metrics

### Usage

Benchmarks are run in docker-compose environment. You can start a benchmark with
```bigquery
make benchmark_run
```
To run benchmarks of old AVR you need to put .deb file with AVR into /bin and name it `avr.deb`
and do the same with dependencies but with `deps.deb` file name. We are running AVR benchmarks on Ubutnu:xenial.

### Configuration

Benchmarks are configurated via .test_env file commited into this repository.
File should look something like this:
```bigquery
export SOCKET_DIR=/tmp
export UNIQUE_DIMENSION_PERCENTAGE=0
export DIMENSION_SIZE=4
export METRICS_PER_MINUTE=100000
export DURATION=10m
export PROMETHEUS_PORT_ADVANCED_METRICS=2112
export PROMETHEUS_PORT_GENERATOR=2113
export NATS_PORT=4222
export SIMPLE_BENCHMARK=true
```

- SOCKET_DIR - name of directory where sockets are going to be put on your machine. It is recommended to leave it as it is
- UNIQUE_DIMENSION_PERCENTAGE - percentage of how often a message going to have one dimension randomly generated instead of taking it from a pool
- DIMENSION_SIZE - how many letters/numbers to add to each string dimension value
- METRICS_PER_MINUTE - how many metric sets to generate each minute. One metric set means one part of message separated by ';' as per advanced metrics contract. Example:
  ```
  "/ c8 GET 1 4e 173 DEV_top_http app_www_variable localFS_app_www_variable 0100007fffff00000000000000000000 4   localhost 0 0 0 0     PASSED  localFS_gw_app_www_variable      0   web http 4e 173"
  ```
- PROMETHEUS_PORT_ADVANCED_METRICS - which port to use for prometheus scraping in advanced metrics. If you change it remember to update prometheus.yml file
- PROMETHEUS_PORT_GENERATOR - which port to use for prometheus scraping in the generator. If you change it remember to update prometheus.yml file
- NATS_PORT - which port to use for NATS communication
- SIMPLE_BENCHMARK - benchmarking has two modes. This value set to true is going to mean that each dimension value will be pulled from a small number of pre-selected strings. 
  If set to false each dimension value will be randomly generated and the pool will be much bigger.

### Setting up external machine for benchmarking

Use setup_external_ubuntu.sh script

### Prometheus and benchmarking metrics

Benchmarks start a prometheus docker that exposes its ui on `http://localhost:9090/`. <br />
Metrics from benchmarks are prefixed with `generator_` and `avr_`
