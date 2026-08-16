# Learning, Community & Commercial — StatGate App 7 (Go backend)
# Build context = repo root so the local statgate-lib replace resolves.
FROM golang:1.25 AS build
WORKDIR /src
COPY statgate-lib ./statgate-lib
COPY learning-crm/backend ./learning-crm/backend
WORKDIR /src/learning-crm/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/learning-crm ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/learning-crm /learning-crm
EXPOSE 8102
ENTRYPOINT ["/learning-crm"]