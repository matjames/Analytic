# Knowledge Portal — StatGate App 2 (Go backend)
# Build context = repo root so the local ../../go.mod replace resolves.
FROM golang:1.25 AS build
WORKDIR /src
COPY statgate-lib ./statgate-lib
COPY knowledge-portal/backend ./knowledge-portal/backend
WORKDIR /src/knowledge-portal/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/knowledge-portal ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/knowledge-portal /knowledge-portal
# Public Open Data Portal UI (no-build HTML/JS) mounted onto the binary.
COPY knowledge-portal/frontend /frontend
ENV KNOWLEDGE_FRONTEND_DIR=/frontend
EXPOSE 8099
ENTRYPOINT ["/knowledge-portal"]
