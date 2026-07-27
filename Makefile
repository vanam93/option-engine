.PHONY: build test run docker-up docker-down lint tidy

build:
	go build -o bin/option-engine ./cmd/server

test:
	go test -race -cover ./...

run:
	go run ./cmd/server -config configs/config.yaml

docker-up:
	docker compose -f deployments/docker/docker-compose.yml up --build -d

docker-down:
	docker compose -f deployments/docker/docker-compose.yml down

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
