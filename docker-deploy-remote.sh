#!/usr/bin/env bash

set -euo pipefail

IMAGE_REF="${1:-${CLI_PROXY_IMAGE:-}}"
BASE_DIR="${CLI_PROXY_BASE_DIR:-$HOME/cliproxyapi}"
CONTAINER_NAME="${CLI_PROXY_CONTAINER_NAME:-cli-proxy-api}"

if [[ -z "${IMAGE_REF}" ]]; then
  echo "Usage: $0 <image-ref>" >&2
  echo "Example: $0 docker.io/example/cli-proxy-api:latest" >&2
  exit 1
fi

mkdir -p "${BASE_DIR}/config" "${BASE_DIR}/data" "${BASE_DIR}/auths"

docker_env_args=()
append_env_arg() {
  local name="$1"
  local value="${!name:-}"
  if [[ -n "${value}" ]]; then
    docker_env_args+=(-e "${name}=${value}")
  fi
}

append_env_arg PGSTORE_DSN
append_env_arg PGSTORE_SCHEMA
append_env_arg MYSQLSTORE_DSN
append_env_arg MYSQLSTORE_DATABASE
append_env_arg MANAGEMENT_PANEL_GITHUB_REPOSITORY
append_env_arg MANAGEMENT_BUNDLED_ASSET_MODE

if [[ -n "${PGSTORE_DSN:-}" && -z "${PGSTORE_LOCAL_PATH:-}" ]]; then
  PGSTORE_LOCAL_PATH=/CLIProxyAPI/data
fi
if [[ -n "${MYSQLSTORE_DSN:-}" && -z "${MYSQLSTORE_LOCAL_PATH:-}" ]]; then
  MYSQLSTORE_LOCAL_PATH=/CLIProxyAPI/data
fi

append_env_arg PGSTORE_LOCAL_PATH
append_env_arg MYSQLSTORE_LOCAL_PATH

if docker ps -a --format '{{.Names}}' | grep -Fxq "${CONTAINER_NAME}"; then
  echo "Container ${CONTAINER_NAME} already exists." >&2
  echo "Remove it first or rerun with RECREATE=1." >&2
  if [[ "${RECREATE:-0}" != "1" ]]; then
    exit 1
  fi
  docker rm -f "${CONTAINER_NAME}"
fi

docker pull "${IMAGE_REF}"

docker run -d \
  --name "${CONTAINER_NAME}" \
  --restart unless-stopped \
  -p "${CLI_PROXY_PORT:-8317}:8317" \
  -p "${CLI_PROXY_AUTH_CALLBACK_PORT_1:-8085}:8085" \
  -p "${CLI_PROXY_AUTH_CALLBACK_PORT_2:-1455}:1455" \
  -p "${CLI_PROXY_AUTH_CALLBACK_PORT_3:-54545}:54545" \
  -p "${CLI_PROXY_AUTH_CALLBACK_PORT_4:-51121}:51121" \
  -p "${CLI_PROXY_AUTH_CALLBACK_PORT_5:-11451}:11451" \
  -e TZ="${TZ:-Asia/Shanghai}" \
  -e DEPLOY="${DEPLOY:-}" \
  -e WRITABLE_PATH=/CLIProxyAPI/data \
  -e CLIPROXY_CONFIG_PATH=/CLIProxyAPI/config/config.yaml \
  "${docker_env_args[@]}" \
  -v "${BASE_DIR}/config:/CLIProxyAPI/config" \
  -v "${BASE_DIR}/data:/CLIProxyAPI/data" \
  -v "${BASE_DIR}/auths:/root/.cli-proxy-api" \
  "${IMAGE_REF}"

echo
echo "Container started: ${CONTAINER_NAME}"
echo "Config path: ${BASE_DIR}/config/config.yaml"
echo "Auth path: ${BASE_DIR}/auths"
echo "API endpoint: http://$(hostname):${CLI_PROXY_PORT:-8317}"
