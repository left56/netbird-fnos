# ADR 0001: Client wrapper boundary

## Decision

`netbird-fnos-api` is a local management wrapper. It obtains NetBird state by executing the official NetBird CLI and, where supported, requests its structured output. The official client remains responsible for all NetBird networking and identity behavior.

## Consequences

The API has no WireGuard, management, signal, relay, route, DNS, SSH, or policy implementation. Commands use an argument vector, a deadline, and a configurable binary path. Secrets must never be logged or returned. A missing client is represented as an unavailable status, never as a connected client.
