FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ensure modules are consistent with source and go.sum
RUN go mod tidy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/statgate-core ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache wget ca-certificates && adduser -D -u 10001 appuser
COPY --from=build /out/statgate-core /statgate-core
USER appuser
EXPOSE 8080
ENTRYPOINT ["/statgate-core"]
