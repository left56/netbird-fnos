# AGENTS.md

## Project goal

Build a native fnOS package for the official NetBird Linux client.

## Non-negotiable boundaries

- Use the official NetBird client binary without modifying NetBird networking code.
- Do not reimplement WireGuard, NetBird protocols, management services, ACLs, routes, DNS, relay, signal, or peer discovery.
- Do not add NAS-specific features such as shared-folder integration, Docker discovery, service publishing, or automatic port scanning.
- The local UI and API are adapters around supported NetBird client commands and documented local interfaces.
- Prefer structured output such as JSON. Do not parse human-readable CLI text when a structured form exists.
- Do not modify or flush unrelated host routes, policy rules, nftables rules, or iptables rules.
- Any system rule created by this package must be identifiable and removable without affecting other software.

## Runtime target

- fnOS on x86_64
- Debian 12 userspace
- systemd 252
- Linux TUN available at `/dev/net/tun`
- WireGuard kernel module available
- nftables and iptables userspace tools available

## Architecture

- `netbird`: official upstream binary.
- `netbird-fnos-api`: small local management service.
- Web UI: static assets served by the local management service or fnOS application entrypoint.
- systemd owns long-running process supervision.
- fnOS lifecycle scripts install, upgrade, start, stop, and uninstall package-owned resources.

## Security

- Bind the management API to loopback or the fnOS-provided application socket by default.
- Never expose setup keys, pre-shared keys, private keys, or identity files in logs or API responses.
- Never pass secrets through shell interpolation. Use direct process arguments or protected files.
- Validate every user-controlled command option against an allowlist.
- The backend must not provide a generic command-execution endpoint.

## Data ownership

Package-owned files and directories must be documented before implementation. NetBird identity and configuration must survive package upgrades. Uninstall behavior must distinguish application removal from optional identity removal.

## Development workflow

- Keep each milestone independently reviewable.
- Add tests for command construction, redaction, configuration validation, and status parsing.
- Document assumptions that depend on fnOS packaging behavior.
- Do not claim an `.fpk` is installable until it has been built and tested on a real fnOS system.
