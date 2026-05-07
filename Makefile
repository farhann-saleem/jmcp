.PHONY: build test lint fmt install clean integration-test

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/jmcp .

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

install:
	go install .

clean:
	rm -rf bin/

integration-test:
	./scripts/integration_test.sh

