# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
ARG NPM_VERSION=12.0.1
ARG PLAYWRIGHT_VERSION=1.61.1

FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ordercli ./cmd/ordercli

FROM node:24.18.0-bookworm-slim
ARG NPM_VERSION
ARG PLAYWRIGHT_VERSION
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN npm install --global npm@${NPM_VERSION} \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && npx --yes playwright@${PLAYWRIGHT_VERSION} install --with-deps chromium \
    && rm -rf /var/lib/apt/lists/* /tmp/* \
    && useradd --create-home --home-dir /data --uid 10001 ordercli \
    && chown -R ordercli:ordercli /data /ms-playwright
ENV HOME=/data \
    XDG_CONFIG_HOME=/data/config \
    XDG_CACHE_HOME=/data/cache
VOLUME ["/data"]
WORKDIR /data
COPY --from=build /out/ordercli /usr/local/bin/ordercli
USER ordercli
ENTRYPOINT ["ordercli"]
CMD ["--help"]
