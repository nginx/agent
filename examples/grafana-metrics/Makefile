OS_RELEASE:=ubuntu
OS_VERSION:=22.04
BASE_IMAGE="docker.io/${OS_RELEASE}:${OS_VERSION}"

help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove docker containers and agent package
	docker-compose down
	rm -rf ./build

build: ## Build agent package
	mkdir -p ../../build
	cd ../../ && GOWORK=off CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o ./build/nginx-agent
	cd ../../ && nfpm pkg --config ./scripts/.local-nfpm.yaml --packager deb --target ./build/nginx-agent.deb

run: build ## Start docker containers
	docker-compose up --build
