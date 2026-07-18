# Architecture Baseline

## 1. Purpose

`netbird-fnos` packages the official NetBird Linux client as a native fnOS application. The package provides local installation, lifecycle management, configuration, status inspection, diagnostics, and a small management UI.

The project is an integration layer. It is not a fork or alternative implementation of NetBird.

## 2. Scope

### Included

- Package and upgrade the official NetBird client binary.
- Register and supervise the local NetBird daemon.
- Connect and disconnect the local peer using supported NetBird client operations.
- Display local peer status, version, connection state, peers, routes, diagnostics, and logs when exposed by official client interfaces.
- Configure supported local client options.
- Preserve NetBird identity and configuration across package upgrades.
- Build an fnOS `.fpk` artifact.

### Excluded

- NetBird management-server administration.
- ACL or policy editing.
- Setup-key creation.
- User and group management.
- Replacement of the official NetBird dashboard.
- Custom peer discovery, shared-folder integration, Docker discovery, automatic port scans, or service publication.
- Reimplementation of NetBird, WireGuard, routing, DNS, relay, signal, or management protocols.

## 3. Runtime assumptions

The first supported target is fnOS with:

- x86_64 architecture;
- Debian 12 userspace;
- systemd 252;
- `/dev/net/tun` available;
- WireGuard kernel module loaded;
- `iproute2`, `nftables`, and `iptables` installed;
- IPv4 forwarding available when the user enables route or exit-node functions.

These assumptions must be checked by installation diagnostics rather than silently assumed.

## 4. Components

```text
fnOS application UI
        |
        | local authenticated HTTP or Unix-socket API
        v
netbird-fnos-api
        |
        | allowlisted official CLI operations
        v
official netbird binary
        |
        v
NetBird daemon / TUN / WireGuard / host networking
```

### 4.1 Official NetBird binary

The upstream binary remains responsible for all NetBird networking behavior. The package pins a version and verifies its checksum during the build. Runtime downloading is not the default distribution model.

### 4.2 Management API

The Go management service provides a narrow interface for the fnOS UI. It may:

- execute explicitly allowlisted NetBird commands;
- read structured status output;
- request daemon lifecycle actions through systemd;
- return redacted logs and diagnostics;
- validate local client settings.

It must not expose arbitrary shell execution or arbitrary command arguments.

### 4.3 Web UI

The UI maps official local-client capabilities into four initial views:

- Dashboard;
- Settings;
- Logs;
- About.

Management-plane features remain in the official NetBird dashboard.

### 4.4 systemd

systemd supervises long-running services. Package scripts should install package-owned unit files rather than relying on an opaque `netbird service install` result.

Initial unit separation:

- `netbird-fnos.service`: official NetBird daemon;
- `netbird-fnos-api.service`: local management API.

Exact NetBird daemon arguments must be derived from the installed upstream version and verified before implementation.

## 5. Proposed local API

Initial endpoints:

```text
GET  /api/health
GET  /api/version
GET  /api/status
GET  /api/config
PUT  /api/config
POST /api/up
POST /api/down
POST /api/restart
GET  /api/logs
GET  /api/debug
```

The API version will be namespaced before the first public release, for example `/api/v1`.

### API rules

- Responses use JSON.
- Secrets are write-only and never returned after storage.
- Errors use stable machine-readable codes.
- Long-running operations use bounded timeouts.
- Concurrent mutating operations are serialized.
- Command output is size-limited.

## 6. Configuration and identity

The package must distinguish three data classes:

1. Package files: replaceable on upgrade.
2. Application settings: management URL and UI-owned preferences.
3. NetBird identity and state: private identity material that must survive upgrade.

The final paths depend on fnOS package conventions and must be validated with an official sample package or packaging documentation. Until then, code must use injected paths and avoid hard-coded assumptions.

Sensitive files require restrictive permissions. Setup keys and pre-shared keys must not be persisted unless the official client requires it. When persistence is necessary, the design must document why and how the secret is protected.

## 7. Lifecycle behavior

### Install

- Validate architecture and required kernel/device capabilities.
- Install packaged binaries and static assets.
- Create package-owned directories with restrictive permissions.
- Install systemd units.
- Reload systemd.
- Do not automatically register a peer without explicit user input.

### Start

- Start the NetBird daemon and management API.
- Report actionable errors if prerequisites are missing.

### Stop

- Stop package-owned services.
- Do not remove identity or network configuration owned by another NetBird installation.

### Upgrade

- Stop services safely.
- Replace package files.
- Preserve configuration and identity.
- migrate package-owned settings when required;
- restart and verify health;
- retain rollback information where fnOS supports it.

### Uninstall

- Stop and disable package-owned units.
- Remove package-owned unit files and application binaries.
- Preserve identity by default unless the user explicitly selects complete data removal.

## 8. Host-network safety

The host may already contain Docker routes, policy-routing tables, and custom rules. Therefore:

- never flush routes, rules, nftables, or iptables globally;
- never assume a routing-table number is unused;
- never delete an object unless ownership can be proven;
- prefer NetBird's own cleanup behavior;
- record any package-created object using deterministic names or comments;
- make cleanup idempotent.

## 9. Security model

The management service is privileged because it controls a network daemon. Required controls:

- bind only to a local fnOS-controlled interface;
- authenticate requests using the fnOS application boundary where available;
- enforce CSRF protection if cookies are used;
- allowlist all command options;
- avoid invoking a shell;
- redact setup keys, pre-shared keys, private keys, tokens, and identity data;
- impose command timeouts and output limits;
- log administrative actions without logging secrets.

A remote network listener is out of scope for the first release.

## 10. Upstream-version strategy

- Pin the NetBird version in one build configuration file.
- Store expected SHA-256 checksums.
- Build artifacts from upstream release assets.
- Do not silently download `latest`.
- Record the bundled NetBird version in the package metadata and About page.
- Test CLI compatibility before each version bump.

## 11. Testing strategy

### Unit tests

- command argument construction;
- configuration validation;
- secret redaction;
- structured status parsing;
- lifecycle error mapping;
- timeout and output-limit behavior.

### Build tests

- Go compilation;
- frontend compilation;
- static analysis;
- package-layout validation;
- checksum verification.

### fnOS integration tests

These require a real fnOS machine or VM:

- install and uninstall `.fpk`;
- service startup and reboot persistence;
- setup-key registration;
- login flow when no setup key is supplied;
- upgrade with identity preservation;
- route and exit-node behavior;
- coexistence with Docker and existing policy routing;
- complete cleanup without collateral network changes.

No release may be described as fnOS-compatible solely because it compiles in CI.

## 12. Open decisions

The following must be resolved before claiming an installable MVP:

- authoritative fnOS package manifest schema;
- application directory variables and persistent-data conventions;
- application UI entrypoint and authentication mechanism;
- whether fnOS permits package-owned systemd units directly;
- exact NetBird daemon invocation for the selected upstream version;
- available structured CLI output and schema stability;
- licensing and redistribution notices required for bundling the official binary.
