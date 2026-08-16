# Geospatial & Remote Sensing — StatGate App 9 (Go backend)
# Build context = repo root so the statgate-lib replace resolves.
FROM golang:1.25 AS build
WORKDIR /src
COPY statgate-lib ./statgate-lib
COPY geointel/backend ./geointel/backend
WORKDIR /src/geointel/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/geointel ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/geointel /geointel
EXPOSE 8103
ENTRYPOINT ["/geointel"]