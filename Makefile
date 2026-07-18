SHELL := /usr/bin/env bash
VERSION ?= dev
NETBIRD_VERSION ?= 0.71.4
TARGET_ARCH ?= x86_64
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BIN := app/bin/netbird-fnos-api
NETBIRD_BIN := app/bin/netbird
FPK ?= fnpack

.PHONY: test build netbird frontend verify-frontend verify-static-files check-lifecycle fpk verify-fpk package install-local uninstall-local clean

test:
	go test ./...
	go vet ./...

build:
	mkdir -p $(dir $(BIN))
	go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o $(BIN) ./cmd/netbird-fnos-api

netbird:
	NETBIRD_VERSION=$(NETBIRD_VERSION) TARGET_ARCH=$(TARGET_ARCH) ./scripts/fetch-netbird.sh $(NETBIRD_BIN)

frontend:
	npm --prefix frontend ci
	npm --prefix frontend run build
	$(MAKE) verify-frontend
	rm -rf app/www/*
	cp -R frontend/dist/. app/www/

verify-frontend:
	npm --prefix frontend run verify-gateway-build

verify-static-files:
	NETBIRD_FNOS_VERIFY_BUILT_FRONTEND=1 go test ./internal/api -run '^TestStaticFilesWithBuiltFrontend$$'

check-lifecycle:
	@test -z "$$(find cmd -maxdepth 1 -type f ! -perm -u=x -print)"
	@test -z "$$(grep -rIl "$$(printf '\r')" cmd || true)"
	@for script in cmd/main cmd/*_init cmd/*_callback; do test "$$(head -n 1 "$$script")" = '#!/bin/sh'; sh -n "$$script"; done
	@env -i PATH=/usr/bin:/bin ./cmd/install_init
	@env -i PATH=/usr/bin:/bin ./cmd/install_callback

fpk: build netbird frontend verify-static-files check-lifecycle
	$(FPK) build
	$(MAKE) verify-fpk

verify-fpk:
	@test -f netbird-fnos.fpk
	@file netbird-fnos.fpk
	@for path in manifest config/privilege config/resource cmd/main cmd/install_init cmd/install_callback app.tgz; do tar -tzf netbird-fnos.fpk | grep -qx "$$path"; done
	@tar -xOzf netbird-fnos.fpk manifest | grep -Eq '^version[[:space:]]*=[[:space:]]*0.1.2$$'
	@for script in cmd/main cmd/install_init cmd/install_callback cmd/upgrade_init cmd/upgrade_callback cmd/uninstall_init cmd/uninstall_callback; do tar -tzvf netbird-fnos.fpk "$$script" | grep -Eq '^-rwx'; ! tar -xOzf netbird-fnos.fpk "$$script" | grep -q "$$(printf '\r')"; done
	@tar -xOzf netbird-fnos.fpk app.tgz | tar -tz | grep -qx bin/netbird
	@tar -xOzf netbird-fnos.fpk app.tgz | tar -tvf - bin/netbird | grep -Eq '^-rwx'
	@tar -xOzf netbird-fnos.fpk app.tgz | tar -xzO www/index.html | grep -Eq '/app/netbird-fnos/assets/'
	@! tar -xOzf netbird-fnos.fpk app.tgz | tar -xzO www/index.html | grep -Eq '(src|href)="(\./assets/|/assets/)'
	@tmpdir="$$(mktemp -d)"; trap 'rm -rf "$$tmpdir"' EXIT; mkdir "$$tmpdir/app"; tar -xzf netbird-fnos.fpk -C "$$tmpdir" ICON.PNG ICON_256.PNG app.tgz; tar -xzf "$$tmpdir/app.tgz" -C "$$tmpdir/app"; cmp -s "$$tmpdir/ICON.PNG" ICON.PNG; cmp -s "$$tmpdir/ICON_256.PNG" ICON_256.PNG; for icon in icon_64.png icon_256.png; do test -f "$$tmpdir/app/ui/images/$$icon"; cmp -s "$$tmpdir/app/ui/images/$$icon" "app/ui/images/$$icon"; done

package: fpk

install-local: fpk
	appcenter-cli install "$$(find . -maxdepth 1 -name '*.fpk' -print -quit)"

uninstall-local:
	appcenter-cli uninstall netbird-fnos

clean:
	rm -f app/bin/netbird-fnos-api app/bin/netbird ./*.fpk
	rm -rf app/www/* frontend/dist frontend/node_modules
