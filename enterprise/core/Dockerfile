FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /statgate-enterprise-core .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /statgate-enterprise-core .
RUN mkdir -p /app/uploads/enterprise

EXPOSE 8096
CMD ["./statgate-enterprise-core"]