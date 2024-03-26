# Go parameters
GOCMD	= go
GOBUILD	= $(GOCMD) build
GOTEST	= $(GOCMD) test
GOTOOL	= $(GOCMD) tool
GORUN	= $(GOCMD) run
GOINST  = $(GOCMD) install
GOGEN   = $(GOCMD) generate
GOVET   = $(GOCMD) vet
GOBIN 	?= $$(go env GOPATH)/bin

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# https://docs.nginx.com/nginx/releases/                                                                                          #
# These images are based on https://github.com/nginxinc/docker-nginx and are NOT recommended for production                       #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# | OS_RELEASE       | OS_VERSION                                | NOTES                                                          |
# | ---------------- | ----------------------------------------- | -------------------------------------------------------------- |
# | amazonlinux      | 2, 2023                                   |                                                                |
# | ubuntu           | 20.04, 22.04                              |                                                                |
# | debian           | bullseye-slim, bookworm-slim 			 |                                                                |
# | centos           | 7                                         |                                                                |
# | redhatenterprise | 7, 8, 9                                   |                                                                |
# | rockylinux       | 8, 9                                      |                                                                |
# | almalinux        | 8, 9                                      |                                                                |
# | alpine           | 3.16, 3.17, 3.18, 3.19                    |                                                                |
# | oraclelinux      | 7, 8, 9                                   |                                                                |
# | suse             | sles12sp5, sle15                          |                                                                |
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
OS_RELEASE  ?= ubuntu
OS_VERSION  ?= 22.04
BASE_IMAGE  = "docker.io/$(OS_RELEASE):$(OS_VERSION)"
IMAGE_TAG   = "agent_$(OS_RELEASE)_$(OS_VERSION)"

BUILD_DIR		:= build
TEST_BUILD_DIR  := build/test
DOCS_DIR        := docs
PROTO_DIR       := proto
BINARY_NAME		:= nginx-agent
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go
DIRS            = $(BUILD_DIR) $(TEST_BUILD_DIR) $(BUILD_DIR)/$(DOCS_DIR) $(BUILD_DIR)/$(DOCS_DIR)/$(PROTO_DIR) $(DOCS_DIR) $(DOCS_DIR)/$(PROTO_DIR)
$(shell mkdir -p $(DIRS))

VERSION 		= "v3.0.0"
COMMIT  		= $(shell git rev-parse --short HEAD)
DATE    		= $(shell date +%F_%H-%M-%S)
LDFLAGS 		= "-w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
DEBUG_LDFLAGS 	= "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

OSS_PACKAGES_REPO 	:= "packages.nginx.org"
PACKAGE_PREFIX 		:= nginx-agent
PACKAGE_NAME 		:= "$(PACKAGE_PREFIX)-$(shell echo $(VERSION) | tr -d 'v')-SNAPSHOT-$(COMMIT)"

MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY ?= 
MOCK_MANAGEMENT_PLANE_LOG_LEVEL ?= INFO
OLD_BENCHMARK_RESULTS_FILE ?= $(TEST_BUILD_DIR)/benchmark.txt

uname_m    := $(shell uname -m)

ifeq ($(uname_m),aarch64)
	OSARCH = arm64
else
	ifeq ($(uname_m),x86_64)
		OSARCH = amd64
	else
		OSARCH = $(uname_m)
	endif
endif

include Makefile.tools
include Makefile.containers

.PHONY: help clean no-local-changes build lint format unit-test integration-test run dev run-mock-management-server generate generate-mocks local-apk-package local-deb-package local-rpm-package

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove build directory
	@rm -rf $(DIRS)
	@echo "🌀 Cleaning Done"

no-local-changes:
	git diff --quiet || { echo "Dependency changes detected. Please commit these before pushing." >&2; exit 1; }

build: ## Build agent executable
	mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	@echo "📦 Build Done"

lint: ## Run linter
	@$(GOVET) ./...
	@$(GORUN) $(GOLANGCILINT) run -c ./.golangci.yml
	@echo "🏯 Linting Done"

format: ## Format code
	@$(GORUN) $(GOFUMPT) -l -w -extra .
	@echo "🏯 Format Done"

unit-test: $(TEST_BUILD_DIR) ## Run unit tests
	@CGO_ENABLED=0 $(GOTEST) -count=1 -coverprofile=$(TEST_BUILD_DIR)/tmp_coverage.out -coverpkg=./... -covermode count ./internal/... ./api/... ./cmd/...
	@cat $(TEST_BUILD_DIR)/tmp_coverage.out | grep -v ".pb.go" | grep -v ".gen.go" | grep -v "fake_" | grep -v "github.com/nginx/agent/v3/test/" > $(TEST_BUILD_DIR)/coverage.out
	@rm $(TEST_BUILD_DIR)/tmp_coverage.out
	@$(GOTOOL) cover -html=$(TEST_BUILD_DIR)/coverage.out -o $(TEST_BUILD_DIR)/coverage.html
	@printf "\nTotal code coverage: " && $(GOTOOL) cover -func=$(TEST_BUILD_DIR)/coverage.out | grep 'total:' | awk '{print $$3}'

race-condition-test: $(TEST_BUILD_DIR) ## Run unit tests with race condition detection
	@CGO_ENABLED=1 $(GOTEST) -race ./internal/... ./api/... ./cmd/...

$(TEST_BUILD_DIR)/coverage.out:
	@$(MAKE) unit-test

.PHONY: coverage
coverage: $(TEST_BUILD_DIR)/coverage.out
	@echo "Checking code coverage"
	@$(GORUN) $(GOTESTCOVERAGE) --config=./.testcoverage.yaml

build-mock-management-plane-http:
	mkdir -p $(BUILD_DIR)/mock-management-plane-http
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/mock-management-plane-http/server test/mock/http/cmd/main.go

build-mock-management-plane-grpc:
	mkdir -p $(BUILD_DIR)/mock-management-plane-grpc
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/mock-management-plane-grpc/server test/mock/grpc/cmd/main.go

integration-test: build-test-package build-mock-management-plane-http build-mock-management-plane-grpc
	TEST_ENV="Container" CONTAINER_OS_TYPE=$(CONTAINER_OS_TYPE) BUILD_TARGET="install-agent-local" \
	PACKAGES_REPO=$(OSS_PACKAGES_REPO) PACKAGE_NAME=$(PACKAGE_NAME) BASE_IMAGE=$(BASE_IMAGE) \
	OS_VERSION=$(OS_VERSION) OS_RELEASE=$(OS_RELEASE) \
	go test -v ./test/integration

performance-test:
	@mkdir -p $(TEST_BUILD_DIR)
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 2m -bench=. -benchmem -run=^# ./internal/service/config > $(TEST_BUILD_DIR)/benchmark.txt
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 2m -bench=. -benchmem -run=^# ./internal/service/instance >> $(TEST_BUILD_DIR)/benchmark.txt
	@cat $(TEST_BUILD_DIR)/benchmark.txt

compare-performance-benchmark-results:
	@$(GORUN) $(BENCHSTAT) $(OLD_BENCHMARK_RESULTS_FILE) $(TEST_BUILD_DIR)/benchmark.txt

run: build ## Run code
	@echo "🏃 Running App"
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run agent executable
	@echo "🚀 Running App"
	$(GORUN) $(PROJECT_DIR)/$(PROJECT_FILE)

race-condition-dev: ## Run agent executable with race condition detection
	@echo "🚀 Running app with race condition detection enabled"
	$(GORUN) -race $(PROJECT_DIR)/$(PROJECT_FILE)

run-mock-management-grpc-server: ## Run mock management plane gRPC server
	@echo "🚀 Running mock management plane gRPC server"
	$(GORUN) test/mock/grpc/cmd/main.go -configDirectory=$(MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY) -logLevel=$(MOCK_MANAGEMENT_PLANE_LOG_LEVEL)

run-mock-management-http-server: ## Run mock management HTTP server
	@echo "🚀 Running mock management plane HTTP server"
	$(GORUN) test/mock/http/cmd/main.go -configDirectory=$(MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY)

generate: ## Generate proto files and server and client stubs from OpenAPI specifications
	@echo "Generating proto files"
	@protoc --go_out=paths=source_relative:./api/grpc/ ./api/grpc/mpi/v1/*.proto --proto_path=./api/grpc/ --go-grpc_out=./api/grpc --doc_out=./$(BUILD_DIR)/$(DOCS_DIR)/$(PROTO_DIR)/ --doc_opt=markdown,protos.md 
	@cp -a ./$(BUILD_DIR)/$(DOCS_DIR)/$(PROTO_DIR)/* ./$(DOCS_DIR)/$(PROTO_DIR)/
	@protoc --go_out=paths=source_relative:. ./api/grpc/**/*.proto
	@echo "Generating Go server and client stubs from OpenAPI specification"
	@$(GORUN) $(OAPICODEGEN) -generate gin -package dataplane ./api/http/dataplane/data-plane-api.yaml > ./api/http/dataplane/data_plane.gen.go
	@$(GORUN) $(OAPICODEGEN) -generate types,client -package dataplane ./api/http/dataplane/data-plane-api.yaml > ./api/http/dataplane/client.gen.go

generate-mocks: ## Regenerate all needed mocks, in order to add new mocks generation add //go:generate to file from witch mocks should be generated
	@echo "Generating mocks"
	@$(GOGEN) ./...
	@echo "\nGenerating mock management plane Go server stubs from OpenAPI specification\n"
	@$(GORUN) $(OAPICODEGEN) -generate gin,types -package http ./test/mock/http/mock-management-plane-api.yaml > ./test/mock/http/mock_management_plane.gen.go

local-apk-package: ## Create local apk package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager apk --target ./build/$(PACKAGE_PREFIX)-$(shell echo $(VERSION) | tr -d 'v')-SNAPSHOT-$(COMMIT).apk;

local-deb-package: ## Create local deb package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager deb --target ./build/$(PACKAGE_PREFIX)-$(shell echo $(VERSION) | tr -d 'v')-SNAPSHOT-$(COMMIT).deb;

local-rpm-package: ## Create local rpm package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager rpm --target ./build/$(PACKAGE_PREFIX)-$(shell echo $(VERSION) | tr -d 'v')-SNAPSHOT-$(COMMIT).rpm;

generate-pgo-profile: build-mock-management-plane-http build-mock-management-plane-grpc
	mv default.pgo profile.pprof
	TEST_ENV="Container" CONTAINER_OS_TYPE=$(CONTAINER_OS_TYPE) BUILD_TARGET="install-agent-local" \
	PACKAGES_REPO=$(OSS_PACKAGES_REPO) PACKAGE_NAME=$(PACKAGE_NAME) BASE_IMAGE=$(BASE_IMAGE) \
	OS_VERSION=$(OS_VERSION) OS_RELEASE=$(OS_RELEASE) \
	$(GOTEST) -v ./test/integration -cpuprofile integration_cpu.pprof
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 2m -bench=. -benchmem -run=^# ./internal/service/config -cpuprofile perf_config_cpu.pprof
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 2m -bench=. -benchmem -run=^# ./internal/service/instance -cpuprofile perf_instance_cpu.pprof
	@$(GOTOOL) pprof -proto perf_config_cpu.pprof perf_instance_cpu.pprof integration_cpu.pprof > default.pgo
	rm perf_config_cpu.pprof perf_instance_cpu.pprof config.test instance.test integration_cpu.pprof integration.test profile.pprof

build-test-package: 
	@if ! [ -e ./build/*.deb ] && [ $(OS_RELEASE) == 'ubuntu' ]; then make local-deb-package; elif ! [ -e ./build/*.rpm ] && [ $(OS_RELEASE) == 'redhatenterprise'  ]; then make local-rpm-package; elif ! [ -e ./build/*.apk ] && [ $(OS_RELEASE) == 'alpine' ]; then make local-apk-package; fi
