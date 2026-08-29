# ─── StatCitizen Multi-Stage Dockerfile ─────────────────────────────────────────

# Stage 1: Build Go Backend
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /statcitizen .

# Stage 2: Production Runtime
FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /statcitizen /app/statcitizen
COPY frontend/ /app/frontend/

ENV STATGATE_ENV=production \
    STATCITIZEN_API_PORT=8097 \
    STATCITIZEN_UI_PORT=3015

EXPOSE 8097
CMD ["/app/statcitizen"]
