help: ## Show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean: ## Remove all .pb.go files
	find . -name "*.pb.go" -not -path "*/vendor/*" -exec rm -f {} \;

lint: ## Run linter
	GOWORK=off go run $(GOLANGCI_LINT) run -c ../scripts/.golangci.yml

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
		--doc_out=./../docs/proto/ \
		--doc_opt=markdown,$$packageName.md \
			proto/$$packageName/*.proto;\
		grep -v '^swagger:' ./../docs/proto/$$packageName.md > ./../docs/proto/tmp-$$packageName.md && mv ./../docs/proto/tmp-$$packageName.md ./../docs/proto/$$packageName.md;\
	done

	protoc \
		-I ./proto/events \
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
		--doc_out=./../docs/proto/ \
		--doc_opt=markdown,proto.md \
		proto/*.proto; \
	grep -v '^swagger:' ./../docs/proto/proto.md > ./../docs/proto/tmp-proto.md && mv ./../docs/proto/tmp-proto.md ./../docs/proto/proto.md
	
	go generate ./...

all-test: unit-test ## Run all tests

unit-test: ## Run unit tests
	go test -v -cover ./...
