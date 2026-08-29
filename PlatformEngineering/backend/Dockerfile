# Build context = repo root so the local statgate-lib replace resolves.
FROM golang:1.22-alpine AS builder
WORKDIR /app/PlatformEngineering/backend

# Shared platform library first, satisfying the replace directive
# (../../statgate-lib relative to /app/PlatformEngineering/backend -> /app/statgate-lib).
COPY statgate-lib /app/statgate-lib

COPY PlatformEngineering/backend/go.mod PlatformEngineering/backend/go.sum ./
RUN go mod download

COPY PlatformEngineering/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o statgate-runops .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/PlatformEngineering/backend/statgate-runops /app/statgate-runops
EXPOSE 8080
CMD ["/app/statgate-runops"]
