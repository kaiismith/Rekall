# ─── Stage 1: Build ───────────────────────────────────────────────────────────
# go.mod requires Go ≥ 1.25; keep this pin in sync with the toolchain directive.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Defensive: the Go image build does not run `npm install`, so husky's prepare
# script is never invoked here. This env var documents intent and protects
# against a future change that pulls in a Node toolchain layer.
ENV HUSKY=0

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download modules first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/rekall-server ./cmd/server

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary and timezone data from builder
COPY --from=builder /app/bin/rekall-server /app/rekall-server
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/rekall-server"]
