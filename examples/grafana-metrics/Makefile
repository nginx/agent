# if you want to build a binary for a different OS change this value https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63#goos-values
# or use the script $(shell uname -s | tr '[:upper:]' '[:lower:]')
OS         := linux
OS_RELEASE := ubuntu
OS_VERSION := 22.04
BASE_IMAGE := "docker.io/${OS_RELEASE}:${OS_VERSION}"

uname_m    := $(shell uname -m)
ifeq ($(uname_m),aarch64)
OSARCH   	            = arm64
else
OSARCH                  = amd64
endif

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove docker containers and agent package
	BASE_IMAGE= docker-compose down
	rm -rf ./build

build: ## Build agent package
	mkdir -p ../../build
	cd ../../ && GOWORK=off CGO_ENABLED=0 GOARCH=${OSARCH} GOOS=${OS} go build -o ./build/nginx-agent
	cd ../../ && ARCH=${OSARCH} nfpm pkg --config ./scripts/.local-nfpm.yaml --packager deb --target ./build/nginx-agent.deb

run: build ## Start docker containers
	BASE_IMAGE=${BASE_IMAGE} docker-compose up --build
