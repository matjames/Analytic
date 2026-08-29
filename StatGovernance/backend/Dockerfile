# Build stage
FROM golang:1.24-alpine AS build
WORKDIR /workspace/StatGovernance/backend
COPY StatGovernance/backend/go.mod StatGovernance/backend/go.sum ./
COPY statgate-lib /workspace/statgate-lib
RUN go mod download
COPY StatGovernance/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o statgovernance-server .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /workspace/StatGovernance/backend/statgovernance-server .
EXPOSE 8080
CMD ["./statgovernance-server"]
