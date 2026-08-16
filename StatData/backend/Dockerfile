# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Copy statgate-lib first for local dependency resolution (repo-root context)
COPY statgate-lib /statgate-lib

# Copy source code
COPY StatData/backend /app

WORKDIR /app
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /statdata-server ./cmd/server

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /app
COPY --from=builder /statdata-server /app/statdata-server
COPY StatData/backend/migrations /app/migrations

EXPOSE 8107

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -f http://localhost:8107/health || exit 1

ENTRYPOINT ["/app/statdata-server"]
