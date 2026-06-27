.PHONY: dev prod build test mock migrate-up migrate-down lint down

include .env
export

dev:
	docker compose --profile dev up

prod:
	docker compose --profile prod up -d

build:
	go build -o bin/bot ./cmd/bot

test:
	go test ./...

mock:
	go generate ./...

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

lint:
	golangci-lint run

down:
	docker compose down
