# syntax=docker/dockerfile:1

# ---- Build ----
FROM golang:1.24-alpine AS builder
WORKDIR /src

# Dependencies first so the module layer is cached across source edits.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-s -w -X main.Version=${VERSION}" \
      -o /out/server ./cmd/server

# ---- Runtime ----
FROM alpine:3.21

# ca-certificates is required for outbound HTTPS; tzdata for correct timestamps.
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app

COPY --from=builder /out/server /app/server
# Migrations are read from disk at startup, relative to the working directory.
COPY migrations ./migrations
COPY config ./config

USER app
EXPOSE 4201

# CONFIG_FILE lets a deployment point at a different config without a rebuild.
ENV CONFIG_FILE=./config/production.yml
ENTRYPOINT ["/bin/sh", "-c", "exec /app/server -config \"$CONFIG_FILE\""]
