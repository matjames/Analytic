FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /statcollect ./cmd/statcollect

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /statcollect /usr/local/bin/statcollect
COPY --from=build /src/frontend /data/frontend
# Server runs migrations from ./migrations relative to WORKDIR (/data).
COPY --from=build /src/migrations /data/migrations

WORKDIR /data
VOLUME ["/data"]
ENV STATCOLLECT_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/statcollect"]
