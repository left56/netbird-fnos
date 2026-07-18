# NetBird for fnOS

Native fnOS application packaging the official NetBird Linux client.

## Scope

The project exposes official NetBird client capabilities through an fnOS-native package and a small local management UI. It does not implement NetBird networking logic or NAS-specific discovery features.

## Status

P0 development scaffold only. It is not production-ready and is not a completed fnOS package.

See [development documentation](docs/development.md), [API contract](docs/api.md), and the [provisional package layout](docs/package-layout.md). The backend deliberately wraps the official NetBird CLI only; it does not implement NetBird networking behavior.
