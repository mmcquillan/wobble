VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test test-short race vet lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o wobble .

test:
	go test ./...

test-short:
	go test -short ./...

race:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed; skipping"

clean:
	rm -f wobble
