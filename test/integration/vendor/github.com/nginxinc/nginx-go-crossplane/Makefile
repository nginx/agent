PACKAGE           = $(notdir $(patsubst %/,%,$(dir $(realpath $(lastword $(MAKEFILE_LIST))))))
OUT_DIR          ?= build
VENDOR_DIR       ?= vendor
RESULTS_DIR		 ?= results
DOCKER_REGISTRY  ?= local
DOCKER_TAG       ?= latest
LINT_BIN := ./bin/golangci-lint

SHELL=/bin/bash
.SHELLFLAGS=-c -eo pipefail

#######################################
## Local set up.
#######################################

.PHONY: init deps deps-upgrade fmt test lint lint-shell gen

init:
	git config core.hooksPath .githooks
	go get golang.org/x/tools/cmd/goimports@v0.1.10
	go get github.com/maxbrunsfeld/counterfeiter/v6@latest
	go get github.com/jstemmer/go-junit-report@latest
	go install golang.org/x/tools/cmd/goimports
	go install github.com/maxbrunsfeld/counterfeiter/v6
	go install github.com/jstemmer/go-junit-report

deps:
	go mod download
	go mod tidy
	go mod verify
	go mod vendor

deps-upgrade:
	GOFLAGS="" go get -u ./...
	$(MAKE) deps

#######################################
## Tests, codegen, lint and format.
#######################################
fmt: $(info Running goimports...)
	@goimports -w -e $$(find . -type f -name '*.go' -not -path "./vendor/*")

test: $(info Running unit tests...)
	mkdir -p $(RESULTS_DIR)
	CGO_ENABLED=1 go test -race -v -cover ./... -coverprofile=$(RESULTS_DIR)/$(PACKAGE)-coverage.out 2>&1 | tee >(go-junit-report > $(RESULTS_DIR)/report.xml)
	@echo "Total code coverage:"
	@go tool cover -func=$(RESULTS_DIR)/$(PACKAGE)-coverage.out | grep 'total:' | tee $(RESULTS_DIR)/anybadge.out
	@go tool cover -html=$(RESULTS_DIR)/$(PACKAGE)-coverage.out -o $(RESULTS_DIR)/coverage.html

test-only-failed: $(info Running unit tests (showing only failed ones with context)...)
	go test -v -race ./... | grep --color -B 45 -A 5 -E '^FAIL.+'

$(LINT_BIN):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.36.0

lint: $(LINT_BIN)
	$(LINT_BIN) run

lint-shell:
	shellcheck -x $$(find . -name "*.sh" -type f -not -path "./vendor/*")

gen:
	go generate -x ./...
	$(MAKE) fmt

#######################################
## Build artifacts for deployment.
#######################################

.PHONY: build-out-dir build build-linux images dev-k8s clean 

build-out-dir:
	@mkdir -p $(OUT_DIR)

# Builds exectuable
build: build-out-dir; $(info Building executable...) @
	CGO_ENABLED=0 go build \
        -v -tags 'release osusergo' \
        -ldflags '-s -w -extldflags "-fno-PIC -static"' \
        -o $(OUT_DIR)/$(PACKAGE) main.go

build-darwin: build-out-dir; $(info Building executable...) @
	CGO_ENABLED=1 go build \
        -v -tags 'release osusergo' \
        -ldflags '-s -w -extldflags "-fno-PIC"' \
        -o $(OUT_DIR)/$(PACKAGE) main.go

build-linux: export GOOS=linux
build-linux: export GOARCH=amd64
build-linux: build

# Removes all build artifacts.
clean: ; $(info Cleaning...) @
	rm -rf $(OUT_DIR)/

# Removes all files that could be downloaded/generated
clean-force: clean; $(info Cleaning everything...) @
	rm -rf $(VENDOR_DIR)/
	rm -rf bin/
	rm -f go.sum
