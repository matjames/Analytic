# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy shared library
COPY statgate-lib /statgate-lib

# Copy service source
COPY StatIoT/backend /app

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /statiot-server ./cmd/server

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /statiot-server .
COPY StatIoT/backend/migrations ./migrations

EXPOSE 8103

ENTRYPOINT ["./statiot-server"]
