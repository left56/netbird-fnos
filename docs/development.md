# Development

Requirements: Go 1.26, Node.js 22, npm, and (for local script linting) ShellCheck.

Run `make test`, `make build`, `make frontend`, and `make package`. The API listens at `127.0.0.1:8080` by default. Configuration uses `NB_FNOS_LISTEN_ADDR`, `NB_FNOS_NETBIRD_BINARY`, and `NB_FNOS_COMMAND_TIMEOUT_SECONDS`; public listener addresses are rejected.

The command adapter is tested with a fake runner. Tests do not need an installed NetBird client.
