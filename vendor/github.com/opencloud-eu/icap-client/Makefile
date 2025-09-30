.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

format:
	go tool gofumpt -extra -w .

.PHONY: clean
clean:
	go clean -i ./...

.PHONY: test
test:
	go test -race -cover -coverprofile coverage.out -timeout 60s .
