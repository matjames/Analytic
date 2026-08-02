FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/statchat-backend ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache wget ca-certificates && adduser -D -u 10001 appuser
COPY --from=build /out/statchat-backend /statchat-backend
USER appuser
EXPOSE 4000
ENTRYPOINT ["/statchat-backend"]