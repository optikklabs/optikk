.PHONY: build test snapshot

build:
	go build -o optikk .

test:
	go test ./...

# Cross-build all archives locally without publishing.
snapshot:
	goreleaser release --snapshot --clean
