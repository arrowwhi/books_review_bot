FROM golang:1.25-alpine AS base
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# ── dev: air для hot reload ──────────────────────────────────────────
FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air", "-c", ".air.toml"]

# ── builder: компиляция обоих бинарников ────────────────────────────
FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/bot ./cmd/bot && \
    CGO_ENABLED=0 GOOS=linux go build -o bin/migrate ./cmd/migrate

# ── prod: оба бинарника в одном образе ──────────────────────────────
FROM alpine:3.19 AS prod
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/bot ./bot
COPY --from=builder /app/bin/migrate ./migrate
ENTRYPOINT ["./bot"]
