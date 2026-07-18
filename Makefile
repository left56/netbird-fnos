SHELL := /usr/bin/env bash
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BIN := app/bin/netbird-fnos-api
FPK ?= fnpack

.PHONY: test build frontend fpk package install-local uninstall-local clean

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

fpk: build frontend
	$(FPK) build

package: fpk

install-local: fpk
	appcenter-cli install "$$(find . -maxdepth 1 -name '*.fpk' -print -quit)"

uninstall-local:
	appcenter-cli uninstall netbird-fnos

clean:
	rm -f app/bin/netbird-fnos-api ./*.fpk
	rm -rf app/www/* frontend/dist frontend/node_modules
