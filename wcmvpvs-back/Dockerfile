# Minimal image to build and run the Go backend
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Preload module metadata and vendor tree
COPY go.mod go.sum ./
COPY vendor ./vendor

# Copy the remaining sources
COPY . .

# Build the API server (CGO is required by mattn/go-sqlite3)
RUN go build -tags '!webui' -o /app/webapi ./cmd/webapi

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/webapi ./webapi
COPY service/database ./service/database

EXPOSE 3000

ENTRYPOINT ["/app/webapi"]
