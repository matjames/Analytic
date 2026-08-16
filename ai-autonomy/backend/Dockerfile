# AI & Autonomy — StatGate App 5 (Go backend)
# Build context = repo root so the local statgate-lib replace resolves.
FROM golang:1.25 AS build
WORKDIR /src
COPY statgate-lib ./statgate-lib
COPY ai-autonomy/backend ./ai-autonomy/backend
WORKDIR /src/ai-autonomy/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ai-autonomy ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ai-autonomy /ai-autonomy
EXPOSE 8101
ENTRYPOINT ["/ai-autonomy"]