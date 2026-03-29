fmt:
	@go fmt

lint:
	@go vet
	@golangci-lint run ./...

flint: fmt lint

test:
	@go test -v ./...