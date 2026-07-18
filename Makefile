SHELL := /usr/bin/env bash
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DIST := dist
BIN := $(DIST)/bin/netbird-fnos-api

.PHONY: test build frontend package clean

test:
	go test ./...
	go vet ./...

build:
	mkdir -p $(dir $(BIN))
	go build -trimpath -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o $(BIN) ./cmd/netbird-fnos-api

frontend:
	npm --prefix frontend ci
	npm --prefix frontend run build

package: build frontend
	rm -rf $(DIST)/package
	mkdir -p $(DIST)/package
	cp -R app manifest scripts $(DIST)/package/
	cp $(BIN) $(DIST)/package/app/bin/
	cp -R frontend/dist/. $(DIST)/package/app/web/
	tar -C $(DIST) -czf $(DIST)/netbird-fnos-provisional.tar.gz package

clean:
	rm -rf $(DIST) frontend/dist frontend/node_modules
