FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
# Copy source
COPY cmd/ ./cmd/

# Download dependencies and build
RUN go mod download 2>/dev/null || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /enterprise-search ./cmd/server/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /enterprise-search .

EXPOSE 8095

ENTRYPOINT ["/app/enterprise-search"]
