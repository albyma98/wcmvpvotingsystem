# syntax=docker/dockerfile:1.6

FROM golang:1.22-bullseye AS build
WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=vendor -o webapi ./cmd/webapi

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata sqlite3 libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /conf

WORKDIR /app
COPY --from=build /app/webapi ./webapi
COPY --from=build /app/service ./service
COPY --from=build /app/demo/config.yml /conf/config.yml

EXPOSE 3000
VOLUME ["/app/service/database"]
CMD ["./webapi"]
