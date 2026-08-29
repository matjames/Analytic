# Build context = repo root so the local statgate-lib replace resolves.
FROM golang:1.22-alpine AS builder
WORKDIR /app/enterprise/search

COPY statgate-lib /app/statgate-lib

COPY enterprise/search/go.mod enterprise/search/go.sum ./
RUN go mod download

COPY enterprise/search/cmd/ ./cmd/
RUN CGO_ENABLED=0 GOOS=linux go build -o /enterprise-search ./cmd/server/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /enterprise-search .

EXPOSE 8095

ENTRYPOINT ["/app/enterprise-search"]
