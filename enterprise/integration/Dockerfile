# Build context = repo root so the local statgate-lib replace resolves.
FROM golang:1.22-alpine AS builder
WORKDIR /app/enterprise/integration

COPY statgate-lib /app/statgate-lib

COPY enterprise/integration/go.mod enterprise/integration/go.sum* ./
RUN go mod download

COPY enterprise/integration/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /statgate-integration-hub .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /statgate-integration-hub .
EXPOSE 8097
CMD ["./statgate-integration-hub"]
