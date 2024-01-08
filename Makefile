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
BINARY_NAME		:= nginx-agent
BINARY			:= $(BUILD_DIR)/$(BINARY_NAME)
PROJECT_DIR		= cmd/agent
PROJECT_FILE	= main.go
IMPORT_MAPPING 	:= ../common/common.yaml:github.com/nginx/agent/v3/internal/apis/http/common

clean:
	@rm -rf ${BUILD_DIR}
	@echo "🌀 Cleaning Done"

no-local-changes:
	git diff --quiet || { echo "Dependency changes detected. Please commit these before pushing." >&2; exit 1; }

build:
	@mkdir -p $@
	@$(GOBUILD) -o $(BINARY) $(PROJECT_DIR)/${PROJECT_FILE}
	@echo "📦 Build Done"

lint:
	GOWORK=off go vet ./...
	GOWORK=off go run github.com/golangci/golangci-lint/cmd/golangci-lint run -c ./scripts/.golangci.yml

format:
	@$(GORUN) mvdan.cc/gofumpt -l -w .
	@echo "🏯 Format Done"

run: $(BINARY)
	@echo "🏃 Running App"
	./${BUILD_DIR}/$(BINARY_NAME)

dev:
	@echo "🚀 Running App"
	$(GORUN) $(PROJECT_DIR)/${PROJECT_FILE}

generate:
	@echo "Generating proto files"
	@protoc --go_out=paths=source_relative:. ./internal/models/*.proto
	@echo "Generating Go server and client stubs from OpenAPI specifications"
	@oapi-codegen -generate types,skip-prune -package common ./internal/apis/http/common/common.yaml > ./internal/apis/http/common/common.gen.go
	@oapi-codegen -generate server,types -package dataplane -import-mapping=$(IMPORT_MAPPING) ./internal/apis/http/dataplane/dataplane-api.yaml > ./internal/apis/http/dataplane/dataplane.gen.go
	@oapi-codegen -generate client -package dataplane -import-mapping=$(IMPORT_MAPPING) ./internal/apis/http/dataplane/dataplane-api.yaml > ./internal/apis/http/dataplane/client.gen.go
