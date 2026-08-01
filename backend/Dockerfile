FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ensure modules are consistent with source and go.sum
RUN go mod tidy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/statgate-core ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/statgate-core /statgate-core
EXPOSE 8080
ENTRYPOINT ["/statgate-core"]
