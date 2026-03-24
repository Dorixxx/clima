FROM node:20-alpine AS management-builder

WORKDIR /ui

COPY mgmt-ui-temp/package.json mgmt-ui-temp/package-lock.json ./

RUN npm ci

COPY mgmt-ui-temp ./

ARG VERSION=dev
ENV VERSION=${VERSION}

RUN npm run build \
    && mkdir -p /out \
    && cp dist/index.html /out/management.html

FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o /out/CLIProxyAPI ./cmd/server/

FROM alpine:3.22.0

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /CLIProxyAPI /CLIProxyAPI/config /CLIProxyAPI/data /root/.cli-proxy-api

COPY --from=builder /out/CLIProxyAPI /CLIProxyAPI/CLIProxyAPI

COPY config.example.yaml /CLIProxyAPI/config.example.yaml
COPY --from=management-builder /out/management.html /CLIProxyAPI/builtin/management.html
COPY docker/entrypoint.sh /usr/local/bin/docker-entrypoint.sh

WORKDIR /CLIProxyAPI

EXPOSE 8317

ENV TZ=Asia/Shanghai \
    WRITABLE_PATH=/CLIProxyAPI/data \
    CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml

RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
