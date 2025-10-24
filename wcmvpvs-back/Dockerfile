# Build Go con CGO per mattn/go-sqlite3
FROM golang:1.22-bookworm AS builder
WORKDIR /app

# Dipendenze build per CGO
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

ENV CGO_ENABLED=1
# Se il tuo codice supporta il flag '!webui', tienilo:
RUN go build -tags '!webui' -o /app/webapi ./cmd/webapi

# Runtime minimale
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/webapi ./webapi

# Se vuoi seedare una directory iniziale per migrazioni o asset statici del back:
# COPY service/database ./service/database

EXPOSE 3000
ENTRYPOINT ["/app/webapi"]
