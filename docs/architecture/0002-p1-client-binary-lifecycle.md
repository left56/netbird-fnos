# P1: NetBird CLI binary lifecycle

## Decision

The FPK always ships one architecture-matched official NetBird CLI at
`app/bin/netbird`. It is the offline fallback and is never overwritten by an
online or uploaded install. Managed binaries live below the package data
directory resolved at runtime from `TRIM_PKGVAR`:

```text
${TRIM_PKGVAR}/netbird/
  versions/<version>/netbird
  versions/<version>/metadata.json
  downloads/
  current -> versions/<version>/netbird
  previous -> versions/<version>/netbird
```

`current` and `previous` are relative symbolic links. A temporary symlink is
renamed into place so readers never observe a partially written link.

## Resolution and trust

`BinaryManager` resolves only these controlled locations, in order:

1. a healthy managed `current` link;
2. the bundled `${TRIM_APPDEST}/bin/netbird` binary;
3. `missing`.

It never accepts an executable path from HTTP input. Every CLI invocation is
obtained from this resolver and has a context deadline. Official release URLs
and their SHA256 checksum files are the trust source. Download accelerators
only transform transport URLs and must receive a URL-encoded full official
URL; their output is accepted only after the official checksum matches.

Manual uploads are staged in a private temporary directory, size-limited,
regular-file checked, safely extracted when compressed, architecture checked,
and version-probed. Uploads without an official checksum are stored as
`upload-unverified` and cannot become active unless an administrator explicitly
sets `allowUnverified` for that request.

## Concurrency and failure handling

A single process-wide mutex serializes installation, selection, rollback and
deletion. Selection validates the target, probes `version`, saves the former
target as `previous`, atomically swaps `current`, and health-checks it with
bounded `netbird version` and `netbird status --check startup` probes. Failed
checks restore `previous`; if that version also fails, `current` is removed and
resolution falls back to the bundled CLI. Current versions cannot be deleted.

The wrapper does not install, elevate, or restart the daemon: fnOS lifecycle
and the official daemon own that privilege boundary.

## Delivery plan

1. Add this document and version/branch baseline.
2. Add reproducible, checksum-verified x86_64 and arm64 FPK build inputs.
3. Implement the constrained Go binary manager and protected API.
4. Add the client-management view and download-source configuration UI.
5. Cover filesystem, trust, authorization and packaged-artifact behavior with
   unit and CI tests; leave fnOS privilege validation to real hardware.

## Permissions assessment

The Web API remains the fnOS package user. This P1 code only manages the
official executable and executes safe fixed subcommands (`version` and status)
with argument arrays; it does not grant root, use `sh -c`, or accept commands
from the browser. The NetBird daemon's need for interface, routing and
firewall privileges is platform-dependent. Before claiming supported operation
on fnOS, validate the official CLI/daemon with the package user's effective
capabilities (in particular `NET_ADMIN` and `NET_RAW`) on hardware, then use
fnOS's narrowly scoped package capability/privileged-entry mechanism if
required. Do not elevate the HTTP API as a workaround.
