FROM golang:1.24.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /src/app /app/app
EXPOSE 9090
CMD ["/app/app"]
