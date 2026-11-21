FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
ARG COMMIT=unknown
ARG BUILT_AT=unknown
ARG VERSION=0.1-dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X atad-project/internal/version.Commit=${COMMIT} -X atad-project/internal/version.BuiltAt=${BUILT_AT} -X atad-project/internal/version.Version=${VERSION}" \
    -o /app ./cmd/app

FROM alpine:3.23
LABEL org.opencontainers.image.title="Git Repository Statistics" \
      org.opencontainers.image.source="https://github.com/BettinaGotiu/atad-project-git-stats"
RUN apk add --no-cache git ca-certificates && adduser -D -u 10001 app
COPY --from=build /app /usr/local/bin/app
ENV GIT_STATS_ROOT=/repos
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]

