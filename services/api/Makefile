.PHONY: run test lint build tidy

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run ./...

build:
	go build ./...

tidy:
	go mod tidy
