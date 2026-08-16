FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o statgate-runops .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/statgate-runops /app/statgate-runops
EXPOSE 8080
CMD ["/app/statgate-runops"]
