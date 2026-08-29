FROM golang:1.22-alpine AS builder

WORKDIR /app/StatOps/backend
COPY statgate-lib /app/statgate-lib
COPY StatOps/backend/go.mod StatOps/backend/go.sum ./
RUN go mod download

COPY StatOps/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o statops-backend .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/StatOps/backend/statops-backend .

EXPOSE 8080
CMD ["./statops-backend"]
