FROM golang:1.24-alpine AS build
WORKDIR /src
COPY StatCollect/go.mod StatCollect/go.sum ./
COPY statgate-lib /statgate-lib
RUN go mod edit -replace github.com/matjames/statgate-lib=/statgate-lib
# Cache the dependency graph separately from application source changes.
RUN go mod download
COPY StatCollect/ ./
# Keep migration packaging explicit so newly added untracked phase migrations
# cannot be omitted from the production image by build-context filtering.
COPY StatCollect/migrations/ /src/migrations/
RUN go build -o /statcollect ./cmd/statcollect

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /statcollect /usr/local/bin/statcollect
COPY --from=build /src/frontend /data/frontend
# Server runs migrations from ./migrations relative to WORKDIR (/data).
COPY --from=build /src/migrations /data/migrations
# Keep a volume-independent copy so persistent /data volumes can receive new
# migrations on startup without being deleted or recreated.
COPY --from=build /src/migrations /opt/statcollect-migrations

WORKDIR /data
VOLUME ["/data"]
ENV STATCOLLECT_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/statcollect"]
