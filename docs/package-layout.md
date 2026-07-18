# fnOS package layout

This repository follows the fnOS package structure: `manifest`, `config/`, `cmd/`, `app/`, `wizard/`, `ICON.PNG`, and `ICON_256.PNG` are at the package root. `cmd/main` implements the documented `start`, `stop`, and `status` actions; installation, upgrade, and uninstall hooks use the matching `cmd/*_init` and `cmd/*_callback` entry points.

The service listens on `${TRIM_APPDEST}/app.sock` and is exposed through the official unified-gateway entry `/app/netbird-fnos`. Runtime files use only `TRIM_APPDEST`, `TRIM_PKGETC`, and `TRIM_PKGVAR`; no systemd unit or `/opt`, `/etc`, or `/var/lib` path is installed by this package.
