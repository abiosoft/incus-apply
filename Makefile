.PHONY: build test clean install gopath-install fmt lint schema doc-setup doc-clean doc doc-incremental doc-serve doc-lint doc-linkcheck doc-spellcheck doc-generate doc-infra doc-publish

SPHINXENV = doc/.sphinx/venv/bin/activate
SPHINXPIP = doc/.sphinx/venv/bin/pip
SPHINXBUILD = doc/.sphinx/venv/bin/sphinx-build

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

install:
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

# Documentation website targets
doc-setup:
	@echo "Setting up documentation build environment"
	python3 -m venv doc/.sphinx/venv
	. $(SPHINXENV) ; $(SPHINXPIP) install --require-virtualenv --upgrade -r doc/.sphinx/requirements.txt
	@mkdir -p doc/html

doc-clean:
	rm -rf doc/.sphinx/.doctrees doc/html

doc-incremental:
	@echo "Building documentation"
	. $(SPHINXENV) ; $(SPHINXBUILD) -c doc/ -b dirhtml doc/ doc/html/ -d doc/.sphinx/.doctrees -w doc/.sphinx/warnings.txt

doc-serve:
	python3 -m http.server 8001 -d doc/html

doc-lint: doc-setup
	. $(SPHINXENV) ; $(SPHINXBUILD) -n -W --keep-going -c doc/ -b dirhtml doc/ doc/html/ -d doc/.sphinx/.doctrees

doc-linkcheck: doc-setup
	. $(SPHINXENV) ; $(SPHINXBUILD) -c doc/ -b linkcheck doc/ doc/html/ -d doc/.sphinx/.doctrees

doc-spellcheck: doc-setup
	. $(SPHINXENV) ; doc/.sphinx/venv/bin/codespell doc README.md docs examples

doc: doc-setup doc-clean doc-incremental

doc-infra:
	incus-apply doc/site.yaml

doc-generate: doc
	cd doc && tar czf site.tar.gz html

doc-publish: doc-generate
	incus file push doc/site.tar.gz incus-apply/tmp/site.tar.gz
	incus exec incus-apply -- /bin/deploy

.DEFAULT_GOAL := build
