# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary (the SQLite driver is pure Go, no CGO needed)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

# Non-root user and a writable data directory for the SQLite database
RUN addgroup -S naturieux && adduser -S -G naturieux naturieux \
    && mkdir -p /data && chown naturieux:naturieux /data

WORKDIR /app

# Application binary and static web assets
COPY --from=build /out/server /app/server
COPY web /app/web

ENV PORT=8080 \
    DB_PATH=/data/naturieux.db \
    MEDIA_DIR=/data/media

EXPOSE 8080
VOLUME ["/data"]
USER naturieux

# Healthcheck hits the existing /health endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]
