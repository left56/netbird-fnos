#!/usr/bin/env bash
set -euo pipefail

PACKAGE_ROOT="${PACKAGE_ROOT:-/opt/netbird-fnos}"
CONFIG_ROOT="${CONFIG_ROOT:-/etc/netbird-fnos}"
STATE_ROOT="${STATE_ROOT:-/var/lib/netbird-fnos}"

ensure_directories() {
  install -d -m 0755 "$PACKAGE_ROOT" "$CONFIG_ROOT" "$STATE_ROOT"
}

service_action() {
  local action="$1"
  if command -v systemctl >/dev/null 2>&1; then systemctl "$action" netbird-fnos-api.service; fi
}

daemon_reload() {
  if command -v systemctl >/dev/null 2>&1; then systemctl daemon-reload; fi
}
