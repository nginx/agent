help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove all .pb.go files
	find . -name "*.pb.go" -exec rm -f {} \;

lint: ## Run linter
	GOWORK=off golangci-lint run -c ../scripts/.golangci.yml

generate: clean ## Generate .pb.go files
	for packageName in "common" "events"; do \
		protoc \
			-I ./proto/$$packageName \
			-I ./proto \
			-I /usr/local/include \
			-I ./vendor/github.com/gogo/protobuf/gogoproto \
			-I ./vendor/github.com/gogo/protobuf/proto \
			--gogofast_out=plugins=grpc,paths=source_relative,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
./proto/$$packageName/ \
			proto/$$packageName/*.proto;\
	done

	protoc \
		-I ./proto \
		-I /usr/local/include \
		-I ./vendor/github.com/gogo/protobuf/gogoproto \
		-I ./vendor/github.com/gogo/protobuf/proto \
		--gogofast_out=plugins=grpc,paths=source_relative,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
./proto/ \
		proto/*.proto
	
	go generate ./sdk/...

all-test: unit-test ## Run all tests

unit-test: ## Run unit tests
	go test -v -cover ./...
