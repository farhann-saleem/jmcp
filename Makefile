.PHONY: build test test-race lint fmt install clean integration-test coverage

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/jmcp .

test:
	go test ./... -count=1

test-race:
	go test -race ./... -count=1

lint:
	go vet ./...
	gofmt -l . | (! grep .) || (echo "gofmt found unformatted files" && exit 1)

fmt:
	gofmt -w .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

install:
	go install -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" .

clean:
	rm -rf bin/ coverage.out

integration-test:
	./scripts/integration_test.sh
