# Go parameters
GOCMD	= go
GOBUILD	= $(GOCMD) build
GOTEST	= $(GOCMD) test
GOTOOL	= $(GOCMD) tool
GORUN	= ${GOCMD} run
GOINST  = ${GOCMD} install
GOGEN   = ${GOCMD} generate
GOVET   = ${GOCMD} vet

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
BASE_IMAGE  = "docker.io/${OS_RELEASE}:${OS_VERSION}"
IMAGE_TAG   = "agent_${OS_RELEASE}_${OS_VERSION}"

CURR_DIR        := $(notdir $(shell pwd)).
BUILD_DIR		:= build
DOCS_DIR        := docs
TEST_BUILD_DIR  := build/test
BINARY_NAME		:= nginx-agent
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go

VERSION = "v3.0.0"
COMMIT  = $(shell git rev-parse --short HEAD)
DATE    = $(shell date +%F_%H-%M-%S)
LDFLAGS = "-w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
DEBUG_LDFLAGS = "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

OSS_PACKAGES_REPO := "packages.nginx.org"
PACKAGE_PREFIX := nginx-agent
PACKAGE_NAME := "${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}"


MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY ?= test/config/nginx

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

.PHONY: help clean no-local-changes build lint format unit-test run dev generate generate-mocks install-tools

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove build directory
	@rm -rf ${BUILD_DIR}
	@echo "🌀 Cleaning Done"

no-local-changes:
	git diff --quiet || { echo "Dependency changes detected. Please commit these before pushing." >&2; exit 1; }

build: ## Build agent executable
	mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags=${LDFLAGS} $(PROJECT_DIR)/${PROJECT_FILE}
	@echo "📦 Build Done"

lint: ## Run linter
	@$(GOVET) ./...
	@$(GORUN) $(GOLANGCILINT) run -c ./.golangci.yml
	@echo "🏯 Linting Done"

format: ## Format code
	@$(GORUN) $(FOFUMPT) -l -w -extra .
	@echo "🏯 Format Done"

$(TEST_BUILD_DIR):
	mkdir -p $(TEST_BUILD_DIR)

unit-test: $(TEST_BUILD_DIR) ## Run unit tests
	@CGO_ENABLED=0 $(GOTEST) -count=1 -coverprofile=$(TEST_BUILD_DIR)/tmp_coverage.out -coverpkg=./... -covermode count ./internal/... ./api/... ./cmd/...
	@cat $(TEST_BUILD_DIR)/tmp_coverage.out | grep -v ".pb.go" | grep -v ".gen.go" | grep -v "fake_" > $(TEST_BUILD_DIR)/coverage.out
	@$(GOTOOL) cover -html=$(TEST_BUILD_DIR)/coverage.out -o $(TEST_BUILD_DIR)/coverage.html
	@printf "\nTotal code coverage: " && $(GOTOOL) cover -func=$(TEST_BUILD_DIR)/coverage.out | grep 'total:' | awk '{print $$3}'

integration-test:
	TEST_ENV="Container" CONTAINER_OS_TYPE=${CONTAINER_OS_TYPE} BUILD_TARGET="install-agent-local" \
	PACKAGES_REPO=${OSS_PACKAGES_REPO} PACKAGE_NAME=${PACKAGE_NAME} BASE_IMAGE=${BASE_IMAGE} \
	OS_VERSION=${OS_VERSION} OS_RELEASE=${OS_RELEASE} \
	go test -v ./test/integration

run: build ## Run code
	@echo "🏃 Running App"
	./${BUILD_DIR}/$(BINARY_NAME)

dev: ## Run agent executable
	@echo "🚀 Running App"
	$(GORUN) $(PROJECT_DIR)/${PROJECT_FILE}

run-mock-management-server: ## Run mock management server
	@echo "🚀 Running mock management server"
	$(GORUN) test/cmd/main.go -configDirectory=$(MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY)

generate: ## Generate proto files and server and client stubs from OpenAPI specifications
	@echo "Generating proto files"
	@mkdir -p ./${BUILD_DIR}/$(DOCS_DIR)
	@mkdir -p ./${BUILD_DIR}/$(DOCS_DIR)/proto
	@protoc --go_out=paths=source_relative:./api/grpc/mpi/v1/ ./api/grpc/mpi/v1/*.proto --proto_path=./api/grpc/mpi/v1 --doc_out=./build/docs/proto/ --doc_opt=markdown,protos.md 
	@cp -a ./${BUILD_DIR}/$(DOCS_DIR)/proto/* ./$(DOCS_DIR)/proto/
    # @protoc --go_out=paths=source_relative:. ./api/grpc/**/*.proto
	# @echo "Generating Go server and client stubs from OpenAPI specification"
	# @$(GORUN) $(OAPICODEGEN) -generate gin -package dataplane ./api/http/dataplane/dataplane-api.yaml > ./api/http/dataplane/dataplane.gen.go
	# @$(GORUN) $(OAPICODEGEN) -generate types,client -package dataplane ./api/http/dataplane/dataplane-api.yaml > ./api/http/dataplane/client.gen.go

generate-mocks: ## Regenerate all needed mocks, in order to add new mocks generation add //go:generate to file from witch mocks should be generated
	@echo "Generating mocks"
	@$(GOGEN) ./...
	@echo "\nGenerating mock management plane Go server stubs from OpenAPI specification\n"
	@$(GORUN) $(OAPICODEGEN) -generate gin,types -package mock ./test/mock/mock-management-plane-api.yaml > ./test/mock/mock_management_plane.gen.go

local-apk-package: ## Create local apk package
	@CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags=${LDFLAGS} $(PROJECT_DIR)/${PROJECT_FILE}
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager apk --target ./build/$(PACKAGE_PREFIX)-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-$(COMMIT).apk;

local-deb-package: ## Create local deb package
	@CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags=${LDFLAGS} $(PROJECT_DIR)/${PROJECT_FILE}
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager deb --target ./build/$(PACKAGE_PREFIX)-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-$(COMMIT).deb;

local-rpm-package: ## Create local rpm package
	@CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags=${LDFLAGS} $(PROJECT_DIR)/${PROJECT_FILE}
	ARCH=$(OSARCH) VERSION=$(shell echo $(VERSION) | tr -d 'v') $(GORUN) $(NFPM) pkg --config ./scripts/packages/.local-nfpm.yaml --packager rpm --target ./build/$(PACKAGE_PREFIX)-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-$(COMMIT).rpm;

