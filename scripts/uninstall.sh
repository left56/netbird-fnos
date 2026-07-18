#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"
service_action disable || true
rm -f /etc/systemd/system/netbird-fnos-api.service
daemon_reload
# NetBird client identity data is deliberately retained. Pass PURGE_NETBIRD_FNOS_STATE=1 to remove only wrapper state.
if [[ "${PURGE_NETBIRD_FNOS_STATE:-0}" == "1" ]]; then rm -rf -- "$STATE_ROOT"; fi
