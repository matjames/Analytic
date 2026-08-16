# RMS backend — Phase y convergence onto statgate-lib.
# Build context must be the repository ROOT so the local `replace` of
# github.com/matjames/statgate-lib (../../statgate-lib) resolves.
FROM golang:1.24-alpine AS build
WORKDIR /app

# Shared platform library first, satisfying the replace directive.
COPY statgate-lib/ /app/statgate-lib/

WORKDIR /app/RMS/backend
COPY RMS/backend/go.mod RMS/backend/go.sum ./
RUN go mod download 2>/dev/null || true

COPY RMS/backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o rms-server .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/RMS/backend/rms-server ./rms-server
ENV PORT=8080
EXPOSE 8080
CMD ["./rms-server"]
