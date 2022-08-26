# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Variable Definitions                                                                                            #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
VERSION = $(shell git describe --match "v[0-9]*" --abbrev=0)
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
DOCKER_IMAGE="${OS_RELEASE}:${OS_VERSION}"
DOCKER_TAG=agent_${OS_RELEASE}_${OS_VERSION}

LDFLAGS = "-w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
DEBUG_LDFLAGS = "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

GPG_PUBLIC_KEY := .key

DEB_DISTROS?=ubuntu-jammy-22.04 ubuntu-focal-20.04 ubuntu-bionic-18.04 debian-bullseye-11 debian-buster-10
DEB_ARCHS?=arm64 amd64
RPM_DISTROS?=centos-7-x86_64 redhatenterprise-7-x86_64 redhatenterprise-8-x86_64 redhatenterprise-9-x86_64 amazon-2-x86_64 amazon-latest-x86_64 amazon-2017.09-x86_64 suse-12-x86_64 suse-15-x86_64
FREEBSD_DISTROS?="FreeBSD:12:amd64" "FreeBSD:13:amd64"
APK_VERSIONS?=3.13 3.14 3.15 3.16
APK_ARCHS?=aarch64 x86_64

PACKAGE_PREFIX	   := nginx-agent
PACKAGES_DIR	   := ./build/packages
PACKAGES_REPO	   := "pkgs-test.nginx.com"
AGENT_UPLOADER_KEY := "~/agent-uploader.pem"
UNAME_M	            = $(shell uname -m)
TEST_BUILD_DIR	   := build/test
# override this value if you want to change the architecture. GOOS options here: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
LOCAL_ARCH         := amd64
UPLOAD_URL         := "https://up-ap.nginx.com/"

ifeq ($(uname_m),aarch64)
ARCH   	                = arm64
else
ARCH                    = amd64
endif

$(TEST_BUILD_DIR):
	mkdir -p $(TEST_BUILD_DIR)

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Developer Targets                                                                                               #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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
	cd sdk && make generate && go mod tidy && go mod vendor
	cd test/performance && go mod tidy && go mod vendor
	go mod tidy && go mod vendor && go mod download && go work sync

lint: ## Run linter
	GOWORK=off go vet ./...
	GOWORK=off golangci-lint run -c ./scripts/.golangci.yml
	cd sdk && make lint

format: ## Format code
	go fmt ./...

install-tools: ## Install dependencies in tools.go
	@grep _ ./scripts/tools.go | awk '{print $$2}' | xargs -tI % go install %

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Release Packaging                                                                                               #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
clean-packages: 
	@rm -rf $(PACKAGES_DIR)

$(PACKAGES_DIR):
	@mkdir -p $(PACKAGES_DIR)/deb && mkdir -p $(PACKAGES_DIR)/rpm && mkdir -p $(PACKAGES_DIR)/apk && mkdir -p $(PACKAGES_DIR)/pkg

package: gpg-key $(PACKAGES_DIR) ## Create final packages for all supported distros
	@for distro in $(DEB_DISTROS); do \
		deb_codename=`echo $$distro | cut -d- -f 2`; \
		for arch in $(DEB_ARCHS); do \
			VERSION=$(shell echo ${VERSION} | tr -d 'v')~$${deb_codename} ARCH=$${arch} nfpm pkg --config .nfpm.yaml --packager deb --target ${PACKAGES_DIR}/deb/${PACKAGE_PREFIX}_$(shell echo ${VERSION} | tr -d 'v')~$${deb_codename}_$${arch}.deb; \
		done; \
	done; \

	for distro in $(RPM_DISTROS); do \
		rpm_distro=`echo $$distro | cut -d- -f 1`;  \
		rpm_major=`echo $$distro | cut -d- -f 2`; \
		rpm_codename='na'; \
		if [ "$$rpm_distro" = "centos" ] || [ "$$rpm_distro" = "redhatenterprise" ]; then rpm_codename="el$$rpm_major"; \
		elif [ "$$rpm_distro" = "amazon" ] && [ "$$rpm_major" = "2" ]; then rpm_codename="amzn$$rpm_major"; fi; \
		if [ "$$rpm_distro" = "suse" ]; then rpm_codename="sles$$rpm_major"; fi; \
		if [ "$$rpm_codename" != "na" ]; then \
			VERSION=$(shell echo ${VERSION} | tr -d 'v') ARCH=${ARCH} nfpm pkg --config .nfpm.yaml --packager rpm --target $(PACKAGES_DIR)/rpm/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v').$${rpm_codename}.ngx.${UNAME_M}.rpm; \
		fi; \
	done; \

	for version in $(APK_VERSIONS); do \
		if [ ! -d "$(PACKAGES_DIR)/apk/$${version}" ]; then mkdir $(PACKAGES_DIR)/apk/$${version}; fi; \
		for arch in $(APK_ARCHS); do \
			if [ ! -d "$(PACKAGES_DIR)/apk/$${version}/$${arch}" ]; then mkdir $(PACKAGES_DIR)/apk/$${version}/$${arch}; fi; \
			VERSION=$(shell echo ${VERSION} | tr -d 'v') ARCH=$${arch} nfpm pkg --config .nfpm.yaml --packager apk --target $(PACKAGES_DIR)/apk/$${version}/$${arch}/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v').apk; \
		done; \
	done; \

	# create specific freebsd pkg files
	rm -rf ./build/nginx-agent
	mkdir -p $(PACKAGES_DIR)/pkg/freebsd

	mkdir -p staging/usr/local/bin
	mkdir -p staging/usr/local/etc/nginx-agent
	mkdir -p staging/usr/local/etc/rc.d

	cp nginx-agent.conf staging/usr/local/etc/nginx-agent
	cp scripts/packages/nginx-agent staging/usr/local/etc/rc.d
	cp scripts/packages/postremove.sh staging/+PRE_DEINSTALL
	cp scripts/packages/postinstall.sh staging/+POST_INSTALL
	cp scripts/packages/plist staging

	GOWORK=off CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags=${LDFLAGS} -o ./build/nginx-agent
	cp build/nginx-agent staging/usr/local/bin

	chmod +x staging/usr/local/etc/rc.d/nginx-agent
	VERSION="$(git describe --match 'v[0-9]*' --abbrev=0 | tr -d 'v')" envsubst < scripts/packages/manifest > staging/+MANIFEST

	for freebsd_abi in $(FREEBSD_DISTROS); do \
		mkdir -p $(PACKAGES_DIR)/pkg/freebsd/$${freebsd_abi}; \
		pkg -o ABI=$${freebsd_abi} create \
			-m staging \
			-r staging \
			-p staging/plist \
			-o $(PACKAGES_DIR)/pkg/freebsd/$${freebsd_abi}; \
		# create freebsd pkg repo layout \
		pkg repo $(PACKAGES_DIR)/pkg/freebsd/$${freebsd_abi} .key.rsa; \
	done; \

	rm -rf staging

	echo "DEB packages:"; \
	find $(PACKAGES_DIR)/deb ;\
	echo "RPM packages:"; \
	find $(PACKAGES_DIR)/rpm ;\
	echo "APK packages:"; \
	find $(PACKAGES_DIR)/apk ;\
	echo "PKG packages:"; \
	find $(PACKAGES_DIR)/pkg ;\
	cd $(PACKAGES_DIR) && tar -czvf "./${PACKAGE_PREFIX}.tar.gz" * && cd ../..;

 gpg-key: ## Generate GPG public key
	$$(gpg --import $(NFPM_SIGNING_KEY_FILE)); \
	keyid=$$(gpg --list-keys NGINX | egrep -A1 "^pub" | egrep -v "^pub" | tr -d '[:space:]'); \
	expiry=1y; \
	$$(gpg --quick-set-expire $$keyid $$expiry '*'); \
	# we need to convert the private gpg key to rsa pem format for pkg signing \
	$$(gpg --export-secret-key $$keyid | openpgp2ssh $$keyid > .key.rsa); \
	$$(gpg --output $(GPG_PUBLIC_KEY) --armor --export)

release: ## Publish tarball to the UPLOAD_URL
	echo "Publishing nginx-agent packages to ${UPLOAD_URL}"; \
	curl -XPOST -F "file=@$(PACKAGES_DIR)/${PACKAGE_PREFIX}.tar.gz" -E ${AGENT_UPLOADER_KEY} ${UPLOAD_URL}; \
	curl -XPOST -F "file=@$(GPG_PUBLIC_KEY)" -E ${AGENT_UPLOADER_KEY} ${UPLOAD_URL}; \

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Local Packaging                                                                                                 #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
local-apk-package: ## Create local apk package
	GOWORK=off CGO_ENABLED=0 GOARCH=${LOCAL_ARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	VERSION=$(shell echo ${VERSION} | tr -d 'v') nfpm pkg --config ./scripts/.local-nfpm.yaml --packager apk --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT.apk;

local-deb-package: ## Create local deb package
	GOWORK=off CGO_ENABLED=0 GOARCH=${LOCAL_ARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	VERSION=$(shell echo ${VERSION} | tr -d 'v') nfpm pkg --config ./scripts/.local-nfpm.yaml --packager deb --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT.deb;

local-rpm-package: ## Create local rpm package
	GOWORK=off CGO_ENABLED=0 GOARCH=${LOCAL_ARCH} GOOS=linux go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	VERSION=$(shell echo ${VERSION} | tr -d 'v') nfpm pkg --config ./scripts/.local-nfpm.yaml --packager rpm --target ./build/${PACKAGE_PREFIX}-$(shell echo ${VERSION} | tr -d 'v')-SNAPSHOT.rpm;

local-pkg-package: ## Create local pkg package
	GOWORK=off CGO_ENABLED=0 GOARCH=${LOCAL_ARCH} GOOS=freebsd go build -ldflags=${DEBUG_LDFLAGS} -o ./build/nginx-agent
	docker run -v `pwd`:/nginx-agent/ build-packager:1.0.0

build-pkg-packager-docker: ## Builds pkg packager docker image
	@echo Building Local Packager; \
	DOCKER_BUILDKIT=1 docker build -t build-packager:1.0.0 . --no-cache -f ./scripts/packages/packager/Dockerfile

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Testing                                                                                                         #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
generate-mocks: # Regenerate all needed mocks, in order to add new mocks generation add //go:generate mockgen to file from witch mocks should be generated
	GOWORK=off go generate ./...

test: unit-test performance-test component-test ## Run all tests

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

test-docker-component: ## Run integration tests in docker
	for container in `docker ps -aqf "name=^nginx-agent_"`; do echo && docker ps -f "id=$$container" --format "{{.Image}}" && docker exec $$container ./tmp/component.test -test.v; done

test-component-run: ## Run component tests
	GOWORK=off CGO_ENABLED=0 go test -v ./test/component

# Performance tests
performance-test: ## Run performance tests
	docker run -v ${PWD}:/home/nginx/ --rm nginx-agent-benchmark:1.0.0

test-bench: ## Run benchmark tests
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 5 -timeout 2m -bench=. -benchmem metrics_test.go
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 1 -bench=. -benchmem userWorkFlow_test.go
	cd test/performance && GOWORK=off CGO_ENABLED=0 go test -mod=vendor -count 5 -timeout 2m -bench=. -benchmem plugins_test.go

build-benchmark-docker: ## Build benchmark test docker image for NGINX Plus, need nginx-repo.crt and nginx-repo.key in build directory
	DOCKER_BUILDKIT=1 docker build --no-cache -t nginx-agent-benchmark:1.0.0 \
		--secret id=nginx-crt,src=build/nginx-repo.crt \
		--secret id=nginx-key,src=build/nginx-repo.key \
		-f test/docker/Dockerfile .

# Install tests
test-install: ## Run agent install test
	GOWORK=off CGO_ENABLED=0 go test -v ./test/install

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Cert Generation                                                                                                 #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
certs: ## Generate TLS certificates
	scripts/mtls/gen_cnf.sh ca --cn 'client-ca.local' --state Cork --locality Cork --org NGINX --country IE --out certs/client/conf
	scripts/mtls/gen_cert.sh ca --config certs/client/conf/ca.cnf --out certs/client

	scripts/mtls/gen_cnf.sh intermediate --cn 'client-int.local' --org NGINX --locality Cork --out certs/client/conf
	scripts/mtls/gen_cert.sh intermediate --config certs/client/conf/int.cnf --ca-cert certs/client/ca.crt --ca-key certs/client/ca.key --out certs/client

	scripts/mtls/gen_cnf.sh end-entity --cn 'client.local' --san 'DNS.1=client.local' --out certs/client/conf
	scripts/mtls/gen_cert.sh end-entity --config certs/client/conf/ee.cnf --ca-cert certs/client/int.crt --ca-key certs/client/int.key --out certs/client

	cp certs/client/ee.crt certs/client.crt
	cp certs/client/ee.key certs/client.key

	scripts/mtls/gen_cnf.sh ca --cn 'server-ca.local' --state Cork --locality Cork --org NGINX --country IE --out certs/server/conf
	scripts/mtls/gen_cert.sh ca --config certs/server/conf/ca.cnf --out certs/server

	scripts/mtls/gen_cnf.sh intermediate --cn 'server-int.local' --org NGINX --locality Cork --out certs/server/conf
	scripts/mtls/gen_cert.sh intermediate --config certs/server/conf/int.cnf --ca-cert certs/server/ca.crt --ca-key certs/server/ca.key --out certs/server

	scripts/mtls/gen_cnf.sh end-entity --cn 'tls.example.com' --san 'DNS.1=tls.example.com' --out certs/server/conf
	scripts/mtls/gen_cert.sh end-entity --config certs/server/conf/ee.cnf --ca-cert certs/server/int.crt --ca-key certs/server/int.key --out certs/server

	cp certs/server/ee.crt certs/server.crt
	cp certs/server/ee.key certs/server.key

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# Docker Helper Targets                                                                                           #
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
build-docker: clean local-apk-package # Build agent docker image for NGINX Plus, need nginx-repo.crt and nginx-repo.key in build directory
	@echo Building Docker; \
	DOCKER_BUILDKIT=1 docker build -t ${DOCKER_TAG} . \
		--no-cache -f ./scripts/docker/${OS_RELEASE}/Dockerfile \
		--secret id=nginx-crt,src=build/nginx-repo.crt \
		--secret id=nginx-key,src=build/nginx-repo.key \
		--build-arg AGENT_CONF="$$(cat nginx-agent.conf)" \
		--build-arg DOCKER_IMAGE=${DOCKER_IMAGE} \
		--build-arg PACKAGES_REPO=${PACKAGES_REPO} \
		--build-arg OS_RELEASE=${OS_RELEASE} \
		--build-arg OS_VERSION=${OS_VERSION}

run-docker: ## Run docker container from specified DOCKER_TAG
	@echo Running Docker; \
		docker run ${DOCKER_TAG}
