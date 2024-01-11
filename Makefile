# Go parameters
GOCMD	= go
GOBUILD	= $(GOCMD) build
GOCLEAN	= $(GOCMD) clean
GOTEST	= $(GOCMD) test
GOTOOL	= $(GOCMD) tool
GOGET	= $(GOCMD) get
GOMOD	= $(GOCMD) mod
GOINST	= $(GOCMD) install
GORUN	= ${GOCMD} run

BUILD_DIR		:= build
TEST_BUILD_DIR  := build/test
BINARY_NAME		:= nginx-agent
BINARY			:= $(BUILD_DIR)/$(BINARY_NAME)
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go
IMPORT_MAPPING 	:= ../common/common.yaml:github.com/nginx/agent/v3/internal/api/http/common

include Makefile.tools

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove build directory
	@rm -rf ${BUILD_DIR}
	@echo "🌀 Cleaning Done"

no-local-changes:
	git diff --quiet || { echo "Dependency changes detected. Please commit these before pushing." >&2; exit 1; }

build: ## Build agent executable
	@mkdir -p $@
	@$(GOBUILD) -o $(BINARY) $(PROJECT_DIR)/${PROJECT_FILE}
	@echo "📦 Build Done"

lint: ## Run linter
	GOWORK=off go vet ./...
	GOWORK=off $(GORUN) $(GOLANGCILINT) run -c ./scripts/.golangci.yml

format: ## Format code
	@$(GORUN) $(FOFUMPT) -l -w .
	@echo "🏯 Format Done"

$(TEST_BUILD_DIR):
	mkdir -p $(TEST_BUILD_DIR)

unit-test: $(TEST_BUILD_DIR) ## Run unit tests
	GOWORK=off CGO_ENABLED=0 $(GOTEST) -count=1 -coverprofile=$(TEST_BUILD_DIR)/tmp_coverage.out -coverpkg=./... -covermode count ./...
	cat $(TEST_BUILD_DIR)/tmp_coverage.out | grep -v ".pb.go" | grep -v ".gen.go" | grep -v "mock_" > $(TEST_BUILD_DIR)/coverage.out
	$(GOTOOL) cover -html=$(TEST_BUILD_DIR)/coverage.out -o $(TEST_BUILD_DIR)/coverage.html
	@printf "\nTotal code coverage: " && $(GOTOOL) cover -func=$(TEST_BUILD_DIR)/coverage.out | grep 'total:' | awk '{print $$3}'

run: $(BINARY) ## Run code
	@echo "🏃 Running App"
	./${BUILD_DIR}/$(BINARY_NAME)

dev: ## Run agent executable
	@echo "🚀 Running App"
	$(GORUN) $(PROJECT_DIR)/${PROJECT_FILE}

generate: ## Genenerate proto files and server and client stubs from OpenAPI specifications
	@echo "Generating proto files"
	@protoc --go_out=paths=source_relative:. ./internal/model/**/*.proto
	@echo "Generating Go server and client stubs from OpenAPI specifications"
	@$(GORUN) $(OAPICODEGEN) -generate types,skip-prune -package common ./internal/api/http/common/common.yaml > ./internal/api/http/common/common.gen.go
	@$(GORUN) $(OAPICODEGEN) -generate gin -package dataplane -import-mapping=$(IMPORT_MAPPING) ./internal/api/http/dataplane/dataplane-api.yaml > ./internal/api/http/dataplane/dataplane.gen.go
	@$(GORUN) $(OAPICODEGEN) -generate client -package dataplane -import-mapping=$(IMPORT_MAPPING) ./internal/api/http/dataplane/dataplane-api.yaml > ./internal/api/http/dataplane/client.gen.go

generate-mocks: ## Regenerate all needed mocks, in order to add new mocks generation add //go:generate to file from witch mocks should be generated
	GOWORK=off go generate ./...
