# BPM / Workflow — StatGate App 11 (Go backend)
# Build context = repo root so the statgate-lib replace resolves.
FROM golang:1.25 AS build
WORKDIR /src
COPY statgate-lib ./statgate-lib
COPY bpm-hub/backend ./bpm-hub/backend
WORKDIR /src/bpm-hub/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bpm-hub ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bpm-hub /bpm-hub
EXPOSE 8104
ENTRYPOINT ["/bpm-hub"]