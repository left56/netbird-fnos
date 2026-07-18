#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"
ensure_directories
install -m 0644 "$PACKAGE_ROOT/systemd/netbird-fnos-api.service" /etc/systemd/system/netbird-fnos-api.service
daemon_reload
service_action enable
