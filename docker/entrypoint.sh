#!/bin/sh
set -eu

config_path="${CLIPROXY_CONFIG_PATH:-/CLIProxyAPI/config/config.yaml}"
template_path="${CLIPROXY_CONFIG_TEMPLATE:-/CLIProxyAPI/config.example.yaml}"
builtin_management_path="${CLIPROXY_BUILTIN_MANAGEMENT_PATH:-/CLIProxyAPI/builtin/management.html}"
management_asset_mode="${MANAGEMENT_BUNDLED_ASSET_MODE:-replace}"
static_root="${WRITABLE_PATH:-/CLIProxyAPI/data}/static"
static_override="${MANAGEMENT_STATIC_PATH:-}"

if [ -n "${static_override}" ]; then
  case "${static_override}" in
    */management.html)
      static_management_path="${static_override}"
      ;;
    *)
      static_management_path="${static_override%/}/management.html"
      ;;
  esac
else
  static_management_path="${static_root}/management.html"
fi

mkdir -p /CLIProxyAPI /CLIProxyAPI/config /CLIProxyAPI/logs /root/.cli-proxy-api
mkdir -p "$(dirname "$config_path")"

if [ -n "${WRITABLE_PATH:-}" ]; then
  mkdir -p "${WRITABLE_PATH}"
fi

if [ -d "${config_path}" ]; then
  echo "config path is a directory: ${config_path}" >&2
  exit 1
fi

if [ ! -f "${template_path}" ]; then
  echo "config template not found: ${template_path}" >&2
  exit 1
fi

if [ ! -f "${config_path}" ]; then
  cp "${template_path}" "${config_path}"
  echo "initialized config from template: ${config_path}" >&2
fi

if [ -f "${builtin_management_path}" ]; then
  mkdir -p "$(dirname "$static_management_path")"
  case "${management_asset_mode}" in
    replace)
      cp "${builtin_management_path}" "${static_management_path}"
      echo "installed bundled management asset: ${static_management_path}" >&2
      ;;
    preserve)
      if [ ! -f "${static_management_path}" ]; then
        cp "${builtin_management_path}" "${static_management_path}"
        echo "initialized bundled management asset: ${static_management_path}" >&2
      fi
      ;;
    disable)
      ;;
    *)
      echo "unsupported MANAGEMENT_BUNDLED_ASSET_MODE: ${management_asset_mode}" >&2
      exit 1
      ;;
  esac
fi

exec /CLIProxyAPI/CLIProxyAPI -config "${config_path}" "$@"
