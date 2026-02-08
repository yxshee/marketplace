.PHONY: run test lint build tidy sqlc-generate migrate-up migrate-down

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

sqlc-generate:
	sqlc generate

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
