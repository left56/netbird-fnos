SHELL := /usr/bin/env bash
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BIN := app/bin/netbird-fnos-api
FPK ?= fnpack

.PHONY: test build frontend check-lifecycle fpk verify-fpk package install-local uninstall-local clean

test:
	go test ./...
	go vet ./...

build:
	mkdir -p $(dir $(BIN))
	go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o $(BIN) ./cmd/netbird-fnos-api

frontend:
	npm --prefix frontend ci
	npm --prefix frontend run build
	rm -rf app/www/*
	cp -R frontend/dist/. app/www/

check-lifecycle:
	@test -z "$$(find cmd -maxdepth 1 -type f ! -perm -u=x -print)"
	@test -z "$$(grep -rIl "$$(printf '\r')" cmd || true)"
	@for script in cmd/main cmd/*_init cmd/*_callback; do test "$$(head -n 1 "$$script")" = '#!/bin/sh'; sh -n "$$script"; done
	@env -i PATH=/usr/bin:/bin ./cmd/install_init
	@env -i PATH=/usr/bin:/bin ./cmd/install_callback

fpk: build frontend check-lifecycle
	$(FPK) build
	$(MAKE) verify-fpk

verify-fpk:
	@test -f netbird-fnos.fpk
	@file netbird-fnos.fpk
	@for path in manifest config/privilege config/resource cmd/main cmd/install_init cmd/install_callback app.tgz; do tar -tzf netbird-fnos.fpk | grep -qx "$$path"; done
	@tar -xOzf netbird-fnos.fpk manifest | grep -Eq '^version[[:space:]]*=[[:space:]]*0.1.1$$'
	@for script in cmd/main cmd/install_init cmd/install_callback cmd/upgrade_init cmd/upgrade_callback cmd/uninstall_init cmd/uninstall_callback; do tar -tzvf netbird-fnos.fpk "$$script" | grep -Eq '^-rwx'; ! tar -xOzf netbird-fnos.fpk "$$script" | grep -q "$$(printf '\r')"; done

package: fpk

install-local: fpk
	appcenter-cli install "$$(find . -maxdepth 1 -name '*.fpk' -print -quit)"

uninstall-local:
	appcenter-cli uninstall netbird-fnos

clean:
	rm -f app/bin/netbird-fnos-api ./*.fpk
	rm -rf app/www/* frontend/dist frontend/node_modules
