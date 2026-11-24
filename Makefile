.PHONY: run build test fmt tidy db-up db-down docker-up docker-down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -count=1

fmt:
	go fmt ./...

tidy:
	go mod tidy

db-up:
	docker compose up -d db

db-down:
	docker compose down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
