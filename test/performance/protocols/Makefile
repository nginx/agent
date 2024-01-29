clean: ## Remove all .pb.go files
	find . -name "*.pb.go" -not -path "*/vendor/*" -exec rm -f {} \;

generate: clean ## Generate .pb.go files
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./*.proto
