VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PREFIX ?= /usr/local
BINARY := optikk
PKG := github.com/optikklabs/optikk/cmd
LDFLAGS := -s -w -buildid= -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)

.PHONY: build install test snapshot size clean

build:
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)

test:
	go test ./...

size: build
	ls -lh $(BINARY)

clean:
	rm -f $(BINARY)

# Cross-build all archives locally without publishing.
snapshot:
	goreleaser release --snapshot --clean
