# Development

Requirements: Go 1.26, Node.js 22, npm, ShellCheck, and the official `fnpack` and `appcenter-cli` tools on an fnOS development environment.

Run `make test`, `make build`, `make frontend`, and `make fpk`. `make install-local` and `make uninstall-local` call the official `appcenter-cli`. fnOS starts the Go service through `cmd/main` and exposes it through its unified gateway on a Unix socket.

The command adapter is tested with a fake runner. Tests do not need an installed NetBird client.
