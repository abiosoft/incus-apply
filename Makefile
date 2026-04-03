.PHONY: build test clean install gopath-install fmt lint schema

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

build:
	go build $(LDFLAGS) -o incus-apply ./cmd/incus-apply

test:
	go test ./... -v

clean:
	rm -f incus-apply

install: build
	install -m 0755 incus-apply $(BINDIR)/incus-apply

gopath-install:
	go install $(LDFLAGS) ./cmd/incus-apply

fmt:
	go fmt ./...

lint:
	golangci-lint run

schema:
	@mkdir -p schema
	go run ./cmd/schema-gen > schema/incus-apply.schema.json

.DEFAULT_GOAL := build
