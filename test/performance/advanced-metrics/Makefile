SHELL:=/bin/bash

#uncomment when you want to use go modules
#export GO111MODULE = on
#export GOFLAGS = -mod=vendor

.PHONY: benchmark_build benchmark_run benchmark_stop benchmark_poc_build \
 benchmark_poc_run benchmark_avr_build benchmark_avr_run benchmark_avr_stop

benchmark_build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/agent-with-acr fake_agent/*.go
	CGO_ENABLED=0 GOOS=linux go build -o bin/generator .

benchmark_run: benchmark_build
	source .test_env && \
	docker-compose -f docker-compose.yml up -d --build --remove-orphans

benchmark_stop:
	source .test_env && \
	docker-compose -f docker-compose.yml down

benchmark_poc_build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/agent-with-poc fake_agent_poc/*.go
	CGO_ENABLED=0 GOOS=linux go build -o bin/generator .

benchmark_poc_run: benchmark_poc_build
	source .test_env && \
	docker-compose -f docker-compose-with-poc.yml up --build --remove-orphans

benchmark_avr_build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/avr_harness avr_harness/*.go
	CGO_ENABLED=0 GOOS=linux go build -o bin/generator .

benchmark_avr_run: benchmark_avr_build
	source .test_env && \
	docker-compose -f docker-compose-with-avr.yml up -d --build --remove-orphans

benchmark_avr_stop:
	source .test_env && \
	docker-compose -f docker-compose-with-avr.yml down