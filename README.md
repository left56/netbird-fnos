# NetBird for fnOS

Native fnOS application packaging the official NetBird Linux client.

## Scope

The project exposes official NetBird client capabilities through an fnOS-native package and a small local management UI. It does not implement NetBird networking logic or NAS-specific discovery features.

## Status

P0 development scaffold only. It is not production-ready.

See [development documentation](docs/development.md), [API contract](docs/api.md), and the [fnOS package layout](docs/package-layout.md). The backend deliberately wraps the official NetBird CLI only; it does not implement NetBird networking behavior.

## fnOS packaging

Install the official `fnpack` tool from the [fnOS developer documentation](https://developer.fnnas.com/docs/cli/fnpack/), then run `make fpk` to create the native `.fpk` package. On an fnOS development host, use `make install-local` and `make uninstall-local` to call the official `appcenter-cli`. The Web UI is integrated through the fnOS unified gateway at `/app/netbird-fnos`; no systemd unit is installed.

This package currently targets x86 fnOS because its Go executable is architecture-specific. Test the generated FPK on a compatible fnOS device before release.
