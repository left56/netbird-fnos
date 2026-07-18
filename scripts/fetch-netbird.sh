#!/usr/bin/env bash
set -euo pipefail

# Fetch a pinned official NetBird release.  This script deliberately queries the
# named release asset list instead of following a mutable "latest" URL.
version="${NETBIRD_VERSION:?NETBIRD_VERSION is required}"
target_arch="${TARGET_ARCH:?TARGET_ARCH is required (x86_64 or arm64)}"
destination="${1:?destination is required}"
case "$target_arch" in x86_64) release_arch=amd64 ;; arm64) release_arch=arm64 ;; *) exit 2 ;; esac
tag="v${version#v}"
api="https://api.github.com/repos/netbirdio/netbird/releases/tags/${tag}"
asset="netbird_${version#v}_linux_${release_arch}.tar.gz"
checksums="netbird_${version#v}_checksums.txt"
metadata="$(curl --fail --location --silent --show-error "$api")"
asset_url="$(jq -r --arg name "$asset" '.assets[] | select(.name == $name) | .browser_download_url' <<<"$metadata")"
checksum_url="$(jq -r --arg name "$checksums" '.assets[] | select(.name == $name) | .browser_download_url' <<<"$metadata")"
test "$asset_url" != null && test "$checksum_url" != null
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
curl --fail --location --silent --show-error "$checksum_url" -o "$tmp/checksums.txt"
curl --fail --location --silent --show-error "$asset_url" -o "$tmp/$asset"
grep -E "[[:space:]]${asset}$" "$tmp/checksums.txt" > "$tmp/expected.txt"
(cd "$tmp" && sha256sum -c expected.txt)
tar -xzf "$tmp/$asset" -C "$tmp"
binary="$(find "$tmp" -type f -name netbird -perm -u+x -print -quit)"
test -n "$binary"
file "$binary" | grep -Eq "(x86-64|aarch64|ARM aarch64)"
case "$target_arch" in x86_64) file "$binary" | grep -q 'x86-64' ;; arm64) file "$binary" | grep -Eq 'aarch64|ARM aarch64' ;; esac
"$binary" version
install -Dm755 "$binary" "$destination"
