# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Variable Definitions                                                                                            #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
VERSION = $(shell git describe --match "v[0-9]*" --abbrev=0 --tags)
COMMIT = $(shell git rev-parse --short HEAD)
DATE = $(shell date +%F_%H-%M-%S)

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# https://docs.nginx.com/nginx/releases/                                                                          #
# These images are based on https://github.com/nginxinc/docker-nginx and are NOT recommended for production       #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# | OS_RELEASE       | OS_VERSION                 | NOTES                                                          |
# | ---------------- | -------------------------- | -------------------------------------------------------------- |
# | amazonlinux      | 2                          |                                                                |
# | ubuntu           | 18.04, 20.04, 22.04        |                                                                |
# | debian           | bullseye-slim, buster-slim |                                                                |
# | centos           | 7                          | centos 7 (below 7.4) uses plus-pkgs.nginx.com as PACKAGES_REPO |
# | redhatenterprise | 7, 8, 9                    |                                                                |
# | alpine           | 3.13, 3.14, 3.15, 3.16     |                                                                |
# | oraclelinux      | 7, 8                       |                                                                |
# | suse             | sles12sp5, sle15           |                                                                |
# | freebsd          |                            | Not supported                                                  |
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
OS_RELEASE:=ubuntu
OS_VERSION:=22.04
BASE_IMAGE="docker.io/${OS_RELEASE}:${OS_VERSION}"
IMAGE_TAG=agent_${OS_RELEASE}_${OS_VERSION}

LDFLAGS = "-w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
DEBUG_LDFLAGS = "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

CERTS_DIR          := ./build/certs
PACKAGE_PREFIX     := nginx-agent
PACKAGES_REPO      := "pkgs.nginx.com"
OS                 := $(shell uname -s | tr '[:upper:]' '[:lower:]')
# override this value if you want to change the architecture. GOOS options here: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
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

TEST_BUILD_DIR     := build/test
PACKAGE_NAME       := "${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}"

CERT_CLIENT_CA_CN  := client-ca.local
CERT_CLIENT_INT_CN := client-int.local
CERT_CLIENT_EE_CN  := client-ee.local
CERT_CLIENT_DNS    := client.local
CERT_SERVER_CA_CN  := server-ca.local
CERT_SERVER_INT_CN := server-int.local
CERT_SERVER_EE_CN  := server-ee.local
CERT_SERVER_DNS    := tls.example.com

include Makefile.containers

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Developer Targets                                                                                               #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

show-var-%:
	@{ \
		escaped_v="$(subst ",\",$($*))" ; \
		if [ -n "$$escaped_v" ]; then v="$$escaped_v"; else v="(undefined)"; fi; \
		printf "%-20s %s\n" "$*" "$$v"; \
	}

SHOW_ENV_VARS = VERSION COMMIT DATE OS OSARCH $(CONTAINER_VARS)
show-env: $(addprefix show-var-, $(SHOW_ENV_VARS)) ## Show environment

all: clean build run ## Compile and run code.

clean: ## Remove build directory
	rm -rf ./build

run: ## Run code
	go run -ldflags=${LDFLAGS} main.go

run-debug: ## Run code
	./build/nginx-agent

build: ## Build agent executable
	GOWORK=off CGO_ENABLED=0 go build -ldflags=${LDFLAGS} -o ./build/nginx-agent

deps: ## Update dependencies in vendor folders
	cd sdk && go mod tidy && go mod vendor &&  make generate && go mod tidy && go mod vendor
	cd test/integration && go mod tidy && go mod vendor
	cd test/performance && go mod tidy && go mod vendor
	go mod tidy && go mod vendor && go mod download && go work sync
	make generate-swagger

lint: ## Run linter
	GOWORK=off go vet ./...
	GOWORK=off golangci-lint run -c ./scripts/.golangci.yml
	cd sdk && make lint

format: ## Format code
	go fmt ./... && cd sdk && go fmt ./... && cd ../test/performance && go fmt ./... && cd ../../test/integration && go fmt ./...
	buf format -w ./sdk/proto/

install-tools: ## Install dependencies in tools.go
	@echo "Getting Tools"
	@grep _ ./scripts/tools.go | awk '{print $$2}' | xargs -tI % go get %
	@echo "Installing Tools"
	@grep _ ./scripts/tools.go | awk '{print $$2}' | xargs -tI % go install %

generate-swagger: ## Generates swagger.json from source code
	go run github.com/go-swagger/go-swagger/cmd/swagger generate spec -o ./docs/swagger.json --scan-models

launch-swagger-ui: generate-swagger ## Launch Swagger UI
	go run github.com/go-swagger/go-swagger/cmd/swagger serve ./docs/swagger.json -F=swagger --port=8082 --no-open
	
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Local Packaging                                                                                                 #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
local-apk-package: ## Create local apk package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager apk --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.apk;

local-deb-package: ## Create local deb package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager deb --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.deb;

local-rpm-package: ## Create local rpm package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	ARCH=${OSARCH} VERSION=$(shell echo ${VERSION} | tr -d 'v') go run github.com/goreleaser/nfpm/v2/cmd/nfpm pkg --config ./scripts/.local-nfpm.yaml --packager rpm --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT-${COMMIT}.rpm;

local-txz-package: ## Create local txz package
	GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=freebsd go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	$(CONTAINER_CLITOOL) run -v ${PWD}:/nginx-agent/$(CONTAINER_VOLUME_FLAGS) build-local-packager:1.0.0

txz-packager-image: ## Builds txz packager container image
	@echo Building Local Packager; \
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t build-local-packager:1.0.0 --build-arg package_type=local-package . --no-cache -f ./scripts/packages/packager/Dockerfile

include Makefile.packaging

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Testing                                                                                                         #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
generate-mocks: ## Regenerate all needed mocks, in order to add new mocks generation add //go:generate mockgen to file from witch mocks should be generated
	GOWORK=off go generate ./...

test: unit-test performance-test component-test integration-test ## Run all tests

$(TEST_BUILD_DIR):
	mkdir -p $(TEST_BUILD_DIR)

# Unit tests
unit-test: $(TEST_BUILD_DIR) test-core test-plugins test-sdk test-extensions ## Run unit tests
	echo 'mode: atomic' > $(TEST_BUILD_DIR)/coverage.out
	tail -q -n +2 $(TEST_BUILD_DIR)/*_coverage.out >> $(TEST_BUILD_DIR)/coverage.out
	go tool cover -html=$(TEST_BUILD_DIR)/coverage.out -o $(TEST_BUILD_DIR)/coverage.html
	@printf "\nTotal code coverage: " && go tool cover -func=$(TEST_BUILD_DIR)/coverage.out | grep 'total:' | awk '{print $$3}'

test-core: $(TEST_BUILD_DIR) ## Run core unit tests
	GOWORK=off CGO_ENABLED=0 go test -count=1 -coverprofile=$(TEST_BUILD_DIR)/core_coverage.out -covermode count ./src/core/...

test-plugins: $(TEST_BUILD_DIR) ## Run plugins unit tests
	GOWORK=off CGO_ENABLED=0 go test -count=1 -coverprofile=$(TEST_BUILD_DIR)/plugins_coverage.out -covermode count ./src/plugins/...

test-extensions: $(TEST_BUILD_DIR) ## Run extensions unit tests
	GOWORK=off CGO_ENABLED=0 go test -count=1 -coverprofile=$(TEST_BUILD_DIR)/extensions_coverage.out -covermode count ./src/extensions/...

test-sdk: $(TEST_BUILD_DIR) ## Run sdk unit tests from root directory
	cd sdk && GOWORK=off CGO_ENABLED=0 go test -count=1 -coverprofile=../$(TEST_BUILD_DIR)/sdk_coverage.out -covermode count ./...

# Component tests
component-test: test-component-build test-component-run ## Run component tests

test-component-build: ## Compile component tests
	GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test ./test/component -c -o component.test

test-container-component: ## Run integration tests in container
	for container in ${$(CONTAINER_CLITOOL) ps -aqf "name=^nginx-agent_"}; do echo && $(CONTAINER_CLITOOL) ps -f "id=$$container" --format "{{.Image}}" && $(CONTAINER_CLITOOL) exec $$container ./tmp/component.test -test.v; done

test-component-run: ## Run component tests
	GOWORK=off CGO_ENABLED=0 go test -v ./test/component/...

# Performance tests
performance-test: ## Run performance tests
	$(CONTAINER_CLITOOL) run -v ${PWD}:/home/nginx/$(CONTAINER_VOLUME_FLAGS) --rm nginx-agent-benchmark:1.0.0

integration-test: local-deb-package
	PACKAGE_NAME=${PACKAGE_NAME} BASE_IMAGE=${BASE_IMAGE} go test -v ./test/integration/install
	PACKAGE_NAME=${PACKAGE_NAME} BASE_IMAGE=${BASE_IMAGE} go test -v ./test/integration/api

test-bench: ## Run benchmark tests
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 5 -timeout 2m -bench=. -benchmem metrics_test.go
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 1 -bench=. -benchmem user_workflow_test.go
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 5 -timeout 2m -bench=. -benchmem plugins_test.go

benchmark-image: ## Build benchmark test container image for NGINX Plus, need nginx-repo.crt and nginx-repo.key in build directory
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build --no-cache -t nginx-agent-benchmark:1.0.0 \
		--secret id=nginx-crt,src=build/nginx-repo.crt \
		--secret id=nginx-key,src=build/nginx-repo.key \
		-f test/docker/Dockerfile .

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Cert Generation                                                                                                 #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
certs: ## Generate TLS certificates
	scripts/tls/gen_cnf.sh ca --cn '${CERT_CLIENT_CA_CN}' --state Cork --locality Cork --org NGINX --country IE --out ${CERTS_DIR}/client/conf
	scripts/tls/gen_cert.sh ca --config ${CERTS_DIR}/client/conf/ca.cnf --out ${CERTS_DIR}/client

	scripts/tls/gen_cnf.sh intermediate --cn '${CERT_CLIENT_INT_CN}' --org NGINX --locality Cork --out ${CERTS_DIR}/client/conf
	scripts/tls/gen_cert.sh intermediate --config ${CERTS_DIR}/client/conf/int.cnf --ca-cert ${CERTS_DIR}/client/ca.crt --ca-key ${CERTS_DIR}/client/ca.key --out ${CERTS_DIR}/client

	scripts/tls/gen_cnf.sh end-entity --cn '${CERT_CLIENT_EE_CN}' --san 'DNS.1=${CERT_CLIENT_DNS}' --out ${CERTS_DIR}/client/conf
	scripts/tls/gen_cert.sh end-entity --config ${CERTS_DIR}/client/conf/ee.cnf --ca-cert ${CERTS_DIR}/client/int.crt --ca-key ${CERTS_DIR}/client/int.key --out ${CERTS_DIR}/client

	cp ${CERTS_DIR}/client/ee.crt ${CERTS_DIR}/client.crt
	cp ${CERTS_DIR}/client/ee.key ${CERTS_DIR}/client.key

	scripts/tls/gen_cnf.sh ca --cn '${CERT_SERVER_CA_CN}' --state Cork --locality Cork --org NGINX --country IE --out ${CERTS_DIR}/server/conf
	scripts/tls/gen_cert.sh ca --config ${CERTS_DIR}/server/conf/ca.cnf --out ${CERTS_DIR}/server

	scripts/tls/gen_cnf.sh intermediate --cn '${CERT_SERVER_INT_CN}' --org NGINX --locality Cork --out ${CERTS_DIR}/server/conf
	scripts/tls/gen_cert.sh intermediate --config ${CERTS_DIR}/server/conf/int.cnf --ca-cert ${CERTS_DIR}/server/ca.crt --ca-key ${CERTS_DIR}/server/ca.key --out ${CERTS_DIR}/server

	scripts/tls/gen_cnf.sh end-entity --cn '${CERT_SERVER_EE_CN}' --san 'DNS.1=${CERT_SERVER_DNS}' --out ${CERTS_DIR}/server/conf
	scripts/tls/gen_cert.sh end-entity --config ${CERTS_DIR}/server/conf/ee.cnf --ca-cert ${CERTS_DIR}/server/int.crt --ca-key ${CERTS_DIR}/server/int.key --out ${CERTS_DIR}/server

	cat ${CERTS_DIR}/server/int.crt ${CERTS_DIR}/server/ca.crt > ${CERTS_DIR}/ca.pem

	cp ${CERTS_DIR}/server/ee.crt ${CERTS_DIR}/server.crt
	cp ${CERTS_DIR}/server/ee.key ${CERTS_DIR}/server.key

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Container Image Helper Targets                                                                                  #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
image: ## Build agent container image for NGINX Plus, need nginx-repo.crt and nginx-repo.key in build directory
	@echo Building image with $(CONTAINER_CLITOOL); \
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t ${IMAGE_TAG} . \
		--no-cache -f ./scripts/docker/nginx-plus/${OS_RELEASE}/Dockerfile \
		--secret id=nginx-crt,src=build/nginx-repo.crt \
		--secret id=nginx-key,src=build/nginx-repo.key \
		--build-arg BASE_IMAGE=${BASE_IMAGE} \
		--build-arg PACKAGES_REPO=${PACKAGES_REPO} \
		--build-arg OS_RELEASE=${OS_RELEASE} \
		--build-arg OS_VERSION=${OS_VERSION}

oss-image: ## Build agent container image for NGINX OSS
	@echo Building image with $(CONTAINER_CLITOOL); \
	$(CONTAINER_BUILDENV) $(CONTAINER_CLITOOL) build -t ${IMAGE_TAG} . \
	--no-cache -f ./scripts/docker/nginx-oss/${OS_RELEASE}/Dockerfile \
	--build-arg PACKAGE_NAME=${PACKAGE_NAME} \
	--build-arg BASE_IMAGE=${BASE_IMAGE} \
	--build-arg ENTRY_POINT=./scripts/docker/nginx-oss/entrypoint.sh

run-container: ## Run container from specified IMAGE_TAG
	@echo Running ${IMAGE_TAG} with $(CONTAINER_CLITOOL); \
		$(CONTAINER_CLITOOL) run --mount type=bind,source=${PWD}/nginx-agent.conf,target=/etc/nginx-agent/nginx-agent.conf ${IMAGE_TAG}

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Grafana Example Dashboard Targets                                                                               #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
clean-grafana-example: clean ## Clean example packages and docker
	cd ./examples/grafana-metrics/ && BASE_IMAGE= PACKAGE_NAME= ${CONTAINER_COMPOSE} down

build-grafana-example: local-deb-package ## Build the example of nginx-agent
	cd ./examples/grafana-metrics/ && BASE_IMAGE=${BASE_IMAGE} PACKAGE_NAME=${PACKAGE_NAME} ${CONTAINER_COMPOSE} build

run-grafana-example: ## Start the example of nginx-agent
	cd ./examples/grafana-metrics/ && BASE_IMAGE=${BASE_IMAGE} PACKAGE_NAME=${PACKAGE_NAME} ${CONTAINER_COMPOSE} up
