FROM golang:1.22-alpine AS builder

WORKDIR /app/StatTrust/backend

COPY statgate-lib /app/statgate-lib
COPY StatTrust/backend/go.mod StatTrust/backend/go.sum ./
RUN go mod download

COPY StatTrust/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o stattrust-backend .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/StatTrust/backend/stattrust-backend /app/stattrust-backend

EXPOSE 8080

CMD ["/app/stattrust-backend"]
