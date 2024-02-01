# Go parameters
GOCMD	= go
GOBUILD	= $(GOCMD) build
GOTEST	= $(GOCMD) test
GOTOOL	= $(GOCMD) tool
GORUN	= ${GOCMD} run
GOINST  = ${GOCMD} install
GOGEN   = ${GOCMD} generate
GOVET   = ${GOCMD} vet

BUILD_DIR		:= build
TEST_BUILD_DIR  := build/test
BINARY_NAME		:= nginx-agent
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go
IMPORT_MAPPING 	:= ../common/common.yaml:github.com/nginx/agent/v3/api/http/common

VERSION = "v3.0.0"
COMMIT  = $(shell git rev-parse --short HEAD)
DATE    = $(shell date +%F_%H-%M-%S)
LDFLAGS = "-w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
DEBUG_LDFLAGS = "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
PACKAGE_PREFIX := nginx-agent

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
	@$(GORUN) $(GOLANGCILINT) run -c ./scripts/.golangci.yml
	@echo "🏯 Linting Done"

format: ## Format code
	@$(GORUN) $(FOFUMPT) -l -w .
	@echo "🏯 Format Done"

$(TEST_BUILD_DIR):
	mkdir -p $(TEST_BUILD_DIR)

unit-test: $(TEST_BUILD_DIR) ## Run unit tests
	@CGO_ENABLED=0 $(GOTEST) -count=1 -coverprofile=$(TEST_BUILD_DIR)/tmp_coverage.out -coverpkg=./... -covermode count ./...
	@cat $(TEST_BUILD_DIR)/tmp_coverage.out | grep -v ".pb.go" | grep -v ".gen.go" | grep -v "fake_" > $(TEST_BUILD_DIR)/coverage.out
	@$(GOTOOL) cover -html=$(TEST_BUILD_DIR)/coverage.out -o $(TEST_BUILD_DIR)/coverage.html
	@printf "\nTotal code coverage: " && $(GOTOOL) cover -func=$(TEST_BUILD_DIR)/coverage.out | grep 'total:' | awk '{print $$3}'

run: build ## Run code
	@echo "🏃 Running App"
	./${BUILD_DIR}/$(BINARY_NAME)

dev: ## Run agent executable
	@echo "🚀 Running App"
	$(GORUN) $(PROJECT_DIR)/${PROJECT_FILE}

generate: ## Genenerate proto files and server and client stubs from OpenAPI specifications
	@echo "Generating proto files"
	@protoc --go_out=paths=source_relative:. ./api/grpc/**/*.proto
	@echo "Generating Go server and client stubs from OpenAPI specifications"
	@$(GORUN) $(OAPICODEGEN) -generate types,skip-prune -package common ./api/http/common/common.yaml > ./api/http/common/common.gen.go
	@$(GORUN) $(OAPICODEGEN) -generate gin -package dataplane -import-mapping=$(IMPORT_MAPPING) ./api/http/dataplane/dataplane-api.yaml > ./api/http/dataplane/dataplane.gen.go
	@$(GORUN) $(OAPICODEGEN) -generate types,client -package dataplane -import-mapping=$(IMPORT_MAPPING) ./api/http/dataplane/dataplane-api.yaml > ./api/http/dataplane/client.gen.go

generate-mocks: ## Regenerate all needed mocks, in order to add new mocks generation add //go:generate to file from witch mocks should be generated
	$(GOGEN) ./...

local-apk-package: ## Create local apk package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -pgo=auto -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager apk --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.apk;

local-deb-package: ## Create local deb package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -pgo=auto -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager deb --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.deb;

local-rpm-package: ## Create local rpm package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -pgo=auto -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager rpm --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.rpm;

