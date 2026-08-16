FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stattrust-backend .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/stattrust-backend /app/stattrust-backend

EXPOSE 8080

CMD ["/app/stattrust-backend"]
