FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY statgate-lib /app/statgate-lib
COPY StatFederation/backend /app/StatFederation/backend

WORKDIR /app/StatFederation/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /statfederation-backend ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=builder /statfederation-backend .
COPY StatFederation/backend/migrations ./migrations

EXPOSE 8099
ENV PORT=8099
ENV STATGATE_ENV=production

CMD ["./statfederation-backend"]
