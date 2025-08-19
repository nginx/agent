# Go parameters
GO_VERSION ?= $(shell cat go.mod | grep toolchain | sed 's/toolchain //; s/go//')
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
# | ubuntu           | 22.04, 24.04 25.04                		 |                                                                |
# | debian           | bullseye-slim, bookworm-slim 			 |                                                                |
# | redhatenterprise | 8, 9, 10                                	 |                                                                |
# | rockylinux       | 8, 9, 10                                  |                                                                |
# | almalinux        | 8, 9, 10                                  |                                                                |
# | alpine           | 3.19, 3.20, 3.21 3.22                     |                                                                |
# | oraclelinux      | 8, 9, 10                                  |                                                                |
# | suse             | sle15                          			 |                                                                |
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
OS_RELEASE  ?= ubuntu
OS_VERSION  ?= 24.04
BASE_IMAGE  = "docker.io/$(OS_RELEASE):$(OS_VERSION)"
IMAGE_TAG   = "agent_$(OS_RELEASE)_$(OS_VERSION)"
DOCKERFILE_PATH = "./test/docker/nginx-oss/$(CONTAINER_OS_TYPE)/Dockerfile"
OFFICIAL_IMAGE_DOCKERFILE_PATH = "./test/docker/nginx-official-image/$(CONTAINER_OS_TYPE)/Dockerfile"
IMAGE_PATH ?= "/nginx/agent"
TAG ?= ""

BUILD_DIR		:= build
TEST_BUILD_DIR  := build/test
CERTS_DIR       := build/certs
DOCS_DIR        := docs
PROTO_DIR       := proto
BINARY_NAME		:= nginx-agent
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go
COLLECTOR_PATH  ?= /etc/nginx-agent/opentelemetry-collector-agent.yaml
MANIFEST_DIR	?= /var/lib/nginx-agent
DIRS            = $(BUILD_DIR) $(TEST_BUILD_DIR) $(BUILD_DIR)/$(DOCS_DIR) $(BUILD_DIR)/$(DOCS_DIR)/$(PROTO_DIR)
$(shell mkdir -p $(DIRS))

VERSION ?= $(shell git describe --match "v[0-9]*" --abbrev=0 --tags)
ifeq ($(strip $(VERSION)),)
	VERSION := $(shell curl https://api.github.com/repos/nginx/agent/releases/latest -s | jq .name -r)
endif
COMMIT  		= $(shell git rev-parse --short HEAD)
DATE    		= $(shell date +%F_%H-%M-%S)
LDFLAGS 		= "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
DEBUG_LDFLAGS 	= "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

OSS_PACKAGES_REPO 	:= "packages.nginx.org"
PACKAGE_PREFIX 		:= nginx-agent
PACKAGE_NAME 		:= $(PACKAGE_PREFIX)-$(shell echo $(VERSION) | tr -d 'v')-SNAPSHOT-$(COMMIT)

APK_PACKAGE := ./build/$(PACKAGE_NAME).apk
DEB_PACKAGE := ./build/$(PACKAGE_NAME).deb
RPM_PACKAGE := ./build/$(PACKAGE_NAME).rpm

MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY ?= 
MOCK_MANAGEMENT_PLANE_LOG_LEVEL ?= INFO
MOCK_MANAGEMENT_PLANE_GRPC_ADDRESS ?= 127.0.0.1:0
MOCK_MANAGEMENT_PLANE_API_ADDRESS ?= 127.0.0.1:0
OLD_BENCHMARK_RESULTS_FILE ?= $(TEST_BUILD_DIR)/benchmark.txt

uname_m    := $(shell uname -m)

ifeq ($(uname_m),aarch64)
	OSARCH ?= arm64
else
	ifeq ($(uname_m),x86_64)
		OSARCH ?= amd64
	else
		OSARCH ?= $(uname_m)
	endif
endif

SELECTED_PACKAGE = $(DEB_PACKAGE)
ifeq ($(OS_RELEASE),redhatenterprise)
	SELECTED_PACKAGE = $(RPM_PACKAGE)
else ifeq ($(OS_RELEASE),alpine)
	SELECTED_PACKAGE = $(APK_PACKAGE)
endif

$(APK_PACKAGE):
	@echo "Building .apk package..."
	@$(MAKE) local-apk-package

$(DEB_PACKAGE): 
	@echo "Building .deb package..."
	@$(MAKE) local-deb-package

$(RPM_PACKAGE):
	@echo "Building .rpm package..."
	@$(MAKE) local-rpm-package

include Makefile.tools
include Makefile.containers
include Makefile.packaging

.PHONY: help clean no-local-changes build lint format unit-test integration-test run dev run-mock-management-grpc-server generate generate-mocks local-apk-package local-deb-package local-rpm-package
help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove build directory
	@rm -rf $(DIRS)
	@echo "🌀 Cleaning Done"

no-local-changes:
	git diff --quiet || { echo "Dependency changes detected. Please commit these before pushing." >&2; exit 1; }

build: ## Build agent executable
	@echo "🏗️ Building"
	mkdir -p $(BUILD_DIR)
	GOARCH=$(OSARCH) $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	@echo "📦 Build Done"

lint: ## Run linter
	@$(GOVET) ./...
	@$(GORUN) $(GOLANGCILINT) run -c ./.golangci.yml
	@cd api/grpc && $(GORUN) $(BUF) generate
	@echo "🏯 Linting Done"

format: ## Format code
	@$(GORUN) $(GOFUMPT) -l -w -extra .
	@echo "🏯 Format Done"

unit-test: $(TEST_BUILD_DIR) ## Run unit tests
	@CGO_ENABLED=0 $(GOTEST) -count=1 -coverprofile=$(TEST_BUILD_DIR)/tmp_coverage.out -coverpkg=./... -covermode count ./internal/... ./api/... ./cmd/... ./pkg/...
	@cat $(TEST_BUILD_DIR)/tmp_coverage.out | grep -v ".pb.go" | grep -v ".gen.go" | grep -v ".pb.validate.go" | grep -v "fake_" | grep -v "_utils.go" | grep -v "github.com/nginx/agent/v3/test/" > $(TEST_BUILD_DIR)/coverage.out
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

build-mock-management-plane-grpc:
	mkdir -p $(BUILD_DIR)/mock-management-plane-grpc
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/mock-management-plane-grpc/server test/mock/grpc/cmd/main.go

build-mock-management-otel-collector:
	mkdir -p $(BUILD_DIR)/mock-management-otel-collector
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/mock-management-otel-collector/collector test/mock/collector/mock-collector/main.go

integration-test: $(SELECTED_PACKAGE) build-mock-management-plane-grpc
	TEST_ENV="Container" CONTAINER_OS_TYPE=$(CONTAINER_OS_TYPE) BUILD_TARGET="install-agent-local" CONTAINER_NGINX_IMAGE_REGISTRY=${CONTAINER_NGINX_IMAGE_REGISTRY} \
	PACKAGES_REPO=$(OSS_PACKAGES_REPO) PACKAGE_NAME=$(PACKAGE_NAME) BASE_IMAGE=$(BASE_IMAGE) DOCKERFILE_PATH=$(DOCKERFILE_PATH) IMAGE_PATH=$(IMAGE_PATH) TAG=${IMAGE_TAG} \
	OS_VERSION=$(OS_VERSION) OS_RELEASE=$(OS_RELEASE) \
	go test -v ./test/integration/installuninstall ./test/integration/managementplane ./test/integration/auxiliarycommandserver ./test/integration/nginxless
	
official-image-integration-test: $(SELECTED_PACKAGE) build-mock-management-plane-grpc
	TEST_ENV="Container" CONTAINER_OS_TYPE=$(CONTAINER_OS_TYPE) CONTAINER_NGINX_IMAGE_REGISTRY=${CONTAINER_NGINX_IMAGE_REGISTRY} BUILD_TARGET="install" \
	PACKAGES_REPO=$(OSS_PACKAGES_REPO) TAG=${TAG} PACKAGE_NAME=$(PACKAGE_NAME) BASE_IMAGE=$(BASE_IMAGE) DOCKERFILE_PATH=$(OFFICIAL_IMAGE_DOCKERFILE_PATH) \
	OS_VERSION=$(OS_VERSION) OS_RELEASE=$(OS_RELEASE) IMAGE_PATH=$(IMAGE_PATH) \
	go test -v ./test/integration/managementplane ./test/integration/auxiliarycommandserver

performance-test:
	@mkdir -p $(TEST_BUILD_DIR)
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 6m -bench=. -benchmem -run=^$$ ./... > $(TEST_BUILD_DIR)/benchmark.txt
	@cat $(TEST_BUILD_DIR)/benchmark.txt

compare-performance-benchmark-results:
	@$(GORUN) $(BENCHSTAT) $(OLD_BENCHMARK_RESULTS_FILE) $(TEST_BUILD_DIR)/benchmark.txt

run: build ## Run code
	@echo "🏃 Running App"
	./$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Run agent executable
	@echo "🚀 Running App"
	NGINX_AGENT_COLLECTOR_CONFIG_PATH=$(COLLECTOR_PATH) NGINX_AGENT_MANIFEST_DIR=$(MANIFEST_DIR) $(GORUN) -ldflags=$(DEBUG_LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)

race-condition-dev: ## Run agent executable with race condition detection
	@echo "🏎️ Running app with race condition detection enabled"
	$(GORUN) -race $(PROJECT_DIR)/$(PROJECT_FILE)

run-mock-management-grpc-server: ## Run mock management plane gRPC server
	@echo "🖲️ Running mock management plane gRPC server"
	$(GORUN) test/mock/grpc/cmd/main.go -configDirectory=$(MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY) -logLevel=$(MOCK_MANAGEMENT_PLANE_LOG_LEVEL) -grpcAddress=$(MOCK_MANAGEMENT_PLANE_GRPC_ADDRESS) -apiAddress=$(MOCK_MANAGEMENT_PLANE_API_ADDRESS)


.PHONY: build-test-nginx-plus-and-nap-image
build-test-nginx-plus-and-nap-image:
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t nginx_plus_and_nap_$(IMAGE_TAG) . \
		--no-cache -f ./test/docker/nginx-plus-and-nap/deb/Dockerfile \
		--secret id=nginx-crt,src=$(CERTS_DIR)/nginx-repo.crt \
		--secret id=nginx-key,src=$(CERTS_DIR)/nginx-repo.key \
		--build-arg PACKAGE_NAME=$(PACKAGE_NAME) \
		--build-arg PACKAGES_REPO=$(OSS_PACKAGES_REPO) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg ENTRY_POINT=./test/docker/entrypoint.sh

.PHONY: build-test-plus-image
build-test-plus-image:
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t nginx_plus_$(IMAGE_TAG) . \
		--no-cache -f ./test/docker/nginx-plus/$(CONTAINER_OS_TYPE)/Dockerfile \
		--secret id=nginx-crt,src=$(CERTS_DIR)/nginx-repo.crt \
		--secret id=nginx-key,src=$(CERTS_DIR)/nginx-repo.key \
		--build-arg PACKAGE_NAME=$(PACKAGE_NAME) \
		--build-arg PACKAGES_REPO=$(OSS_PACKAGES_REPO) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg ENTRY_POINT=./test/docker/entrypoint.sh

.PHONY: build-test-oss-image
build-test-oss-image:
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t nginx_oss_$(IMAGE_TAG) . \
		--no-cache -f ./test/docker/nginx-oss/$(CONTAINER_OS_TYPE)/Dockerfile \
		--target install-agent-local \
		--build-arg PACKAGE_NAME=$(PACKAGE_NAME) \
		--build-arg PACKAGES_REPO=$(OSS_PACKAGES_REPO) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg ENTRY_POINT=./test/docker/entrypoint.sh
		
.PHONY: build-mock-management-otel-collector-image
build-mock-management-otel-collector-image: build-mock-management-otel-collector
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t mock-collector . \
		--no-cache -f ./test/mock/collector/mock-collector/Dockerfile

.PHONY: run-mock-management-otel-collector
run-mock-management-otel-collector: ## Run mock management plane OTel collector
	@echo "🚀 Running mock management plane OTel collector"
	AGENT_IMAGE_WITH_NGINX_PLUS=nginx_plus_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_OSS=nginx_oss_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_PLUS_AND_NAP=nginx_plus_and_nap_$(IMAGE_TAG):latest $(CONTAINER_COMPOSE) -f ./test/mock/collector/nginx-plus-and-nap/docker-compose.yaml up -d

.PHONY: stop-mock-management-otel-collector
stop-mock-management-otel-collector: ## Stop running mock management plane OTel collector
	@echo "Stopping mock management plane OTel collector"
	AGENT_IMAGE_WITH_NGINX_PLUS=nginx_plus_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_OSS=nginx_oss_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_PLUS_AND_NAP=nginx_plus_and_nap_$(IMAGE_TAG):latest $(CONTAINER_COMPOSE) -f ./test/mock/collector/nginx-plus-and-nap/docker-compose.yaml down

.PHONY: run-mock-otel-collector-without-nap
run-mock-otel-collector-without-nap:
	@echo "🚀 Running mock management plane OTel collector without NAP"
	AGENT_IMAGE_WITH_NGINX_PLUS=nginx_plus_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_OSS=nginx_oss_$(IMAGE_TAG):latest $(CONTAINER_COMPOSE) -f ./test/mock/collector/docker-compose.yaml up -d

.PHONY: stop-mock-otel-collector-without-nap
stop-mock-otel-collector-without-nap: ## Stop running mock management plane OTel collector
	@echo "Stopping mock management plane OTel collector without NAP"
	AGENT_IMAGE_WITH_NGINX_PLUS=nginx_plus_$(IMAGE_TAG):latest AGENT_IMAGE_WITH_NGINX_OSS=nginx_oss_$(IMAGE_TAG):latest $(CONTAINER_COMPOSE) -f ./test/mock/collector/docker-compose.yaml down

generate: ## Generate golang code
	@echo "🗄️ Generating proto files"
	@cd api/grpc && $(GORUN) $(BUF) generate
	@echo "🗃️ Generating go files"
	@$(GOGEN) ./...

local-apk-package: ## Create local apk package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager apk --target $(APK_PACKAGE);

local-deb-package: ## Create local deb package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager deb --target $(DEB_PACKAGE);

local-rpm-package: ## Create local rpm package
	@CGO_ENABLED=0 GOARCH=$(OSARCH) GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -pgo=default.pgo -ldflags=$(LDFLAGS) $(PROJECT_DIR)/$(PROJECT_FILE)
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager rpm --target $(RPM_PACKAGE);

generate-pgo-profile: build-mock-management-plane-grpc
	mv default.pgo profile.pprof
	TEST_ENV="Container" CONTAINER_OS_TYPE=$(CONTAINER_OS_TYPE) BUILD_TARGET="install-agent-local" \
	PACKAGES_REPO=$(OSS_PACKAGES_REPO) PACKAGE_NAME=$(PACKAGE_NAME) BASE_IMAGE=$(BASE_IMAGE) \
	OS_VERSION=$(OS_VERSION) OS_RELEASE=$(OS_RELEASE) DOCKERFILE_PATH=$(DOCKERFILE_PATH) \
	IMAGE_PATH=$(IMAGE_PATH) TAG=${IMAGE_TAG} CONTAINER_NGINX_IMAGE_REGISTRY=${CONTAINER_NGINX_IMAGE_REGISTRY} \
	$(GOTEST) -v ./test/integration -cpuprofile integration_cpu.pprof
	@CGO_ENABLED=0 $(GOTEST) -count 10 -timeout 5m -bench=. -benchmem -run=^# ./internal/watcher/instance -cpuprofile perf_watcher_cpu.pprof
	@$(GOTOOL) pprof -proto perf_watcher_cpu.pprof integration_cpu.pprof > default.pgo
	rm perf_watcher_cpu.pprof integration_cpu.pprof integration.test profile.pprof

# run under sudo locally
load-test-image: ## Build performance load testing image
	@echo "🚚 Running load tests"
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t $(IMAGE_TAG)_load_test . \
		--no-cache -f ./test/docker/load/Dockerfile \
		--secret id=nginx-crt,src=$(CERTS_DIR)/nginx-repo.crt \
		--secret id=nginx-key,src=$(CERTS_DIR)/nginx-repo.key \
		--build-arg OSARCH=$(OSARCH) \
		--build-arg GO_VERSION=$(GO_VERSION)

run-load-test-image: ## Run performance load testing image
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) run --rm -v $(PWD)/$(BUILD_DIR)/:/agent/$(BUILD_DIR)/ $(IMAGE_TAG)_load_test
