.PHONY: dev prod build test mock migrate docker-migrate lint down docker-build

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

migrate:
	go run ./cmd/migrate

docker-build:
	docker build --target prod -t books-bot .

docker-migrate:
	docker run --rm --env-file .env books-bot ./migrate

lint:
	golangci-lint run

down:
	docker compose down
