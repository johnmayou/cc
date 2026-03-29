fmt:
	@go fmt

lint:
	@go vet
	@golangci-lint run ./...

test:
	@go test -v ./...

test-cli:
	@bash ./test.sh