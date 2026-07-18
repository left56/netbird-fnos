# Repository guidance

This project is a local fnOS wrapper around the official NetBird Linux client.

- The backend may invoke only the official `netbird` CLI; do not implement networking, identity, routing, DNS, SSH, policy, or WireGuard logic.
- Use argument arrays with `os/exec`, apply a context timeout to every CLI call, and never construct shell commands.
- Do not log or return setup keys, pre-shared keys, tokens, private keys, or CLI output that can contain them.
- The API must listen on a loopback address by default. Do not change the default to `0.0.0.0`.
- Keep the P0 scope to the documented wrapper, UI scaffold, package layout, and build tooling.
