# ─── StatCitizen Multi-Stage Dockerfile ─────────────────────────────────────────

# Stage 1: Build Go Backend
FROM golang:1.22-alpine AS builder
WORKDIR /app/StatCitizen/backend
COPY statgate-lib /app/statgate-lib
COPY StatCitizen/backend/go.mod StatCitizen/backend/go.sum* ./
COPY StatCitizen/backend/vendor ./vendor
COPY StatCitizen/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o /statcitizen .

# Stage 2: Production Runtime
FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /statcitizen /app/statcitizen
COPY StatCitizen/frontend/ /app/frontend/

ENV STATGATE_ENV=production \
    STATCITIZEN_API_PORT=8115 \
    STATCITIZEN_UI_PORT=8115

EXPOSE 8115
CMD ["/app/statcitizen"]
