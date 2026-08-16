# StatSpatial backend (Phase y convergence: StatSpatial brought into the platform).
# Build context must be the repository ROOT so the local `replace` of
# github.com/matjames/statgate-lib (../../statgate-lib) resolves.
FROM golang:1.25-alpine AS build
WORKDIR /app

# Shared platform library first, satisfying the replace directive.
COPY statgate-lib/ /app/statgate-lib/

WORKDIR /app/StatSpatial/backend
COPY StatSpatial/backend/go.mod StatSpatial/backend/go.sum ./
RUN go mod download 2>/dev/null || true

COPY StatSpatial/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/statspatial-server ./cmd/server

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/statspatial-server ./statspatial-server
ENV BACKEND_PORT=4200
EXPOSE 4200
CMD ["./statspatial-server"]