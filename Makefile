# ════════════════════════════════════════════════════════════════════
#  NIGHTCRAWLER v7.0 — Makefile
#  Run `make help` for a list of targets.
# ════════════════════════════════════════════════════════════════════

SHELL          := /usr/bin/env bash
.SHELLFLAGS    := -euo pipefail -c
.DEFAULT_GOAL  := help

# ── Build metadata (overridable) ────────────────────────────────────
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X github.com/1607-NetEnginee/NightCrawler/internal/version.Version=$(VERSION) \
  -X github.com/1607-NetEnginee/NightCrawler/internal/version.Commit=$(COMMIT) \
  -X github.com/1607-NetEnginee/NightCrawler/internal/version.BuildDate=$(BUILD_DATE)

BIN_DIR := ./bin
BIN     := $(BIN_DIR)/nightcrawler

# ────────────────────────────────────────────────────────────────────

.PHONY: help
help:                          ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build:                         ## Build the nightcrawler binary into ./bin.
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/nightcrawler
	@echo "✓ built $(BIN)"

.PHONY: install
install:                       ## Install nightcrawler to $$GOPATH/bin.
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/nightcrawler

.PHONY: run
run:                           ## Build and run with --help.
	@$(MAKE) build
	@$(BIN) --help

.PHONY: test
test:                          ## Run unit tests with race detector.
	go test ./... -race -count=1 -timeout=120s

.PHONY: test-cover
test-cover:                    ## Run tests and write cover.out + cover.html.
	go test ./... -race -count=1 -coverprofile=cover.out -covermode=atomic
	go tool cover -html=cover.out -o cover.html
	@echo "✓ coverage report: cover.html"

.PHONY: bench
bench:                         ## Run benchmarks.
	go test -bench=. -benchmem -run=^$$ ./...

.PHONY: lint
lint:                          ## Run gofmt, go vet, and golangci-lint.
	@diff=$$(gofmt -l .); if [ -n "$$diff" ]; then echo "gofmt issues:"; echo "$$diff"; exit 1; fi
	go vet ./...
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run --timeout=5m

.PHONY: fmt
fmt:                           ## Format Go sources.
	gofmt -w .

.PHONY: tidy
tidy:                          ## Run go mod tidy.
	go mod tidy

.PHONY: vuln
vuln:                          ## Run govulncheck.
	@command -v govulncheck >/dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

.PHONY: docker-build
docker-build:                  ## Build the Docker image (tag: nightcrawler:dev).
	docker build -t nightcrawler:dev \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -f deployments/docker/Dockerfile .

.PHONY: docker-run
docker-run:                    ## Run the Docker image with --help.
	docker run --rm nightcrawler:dev --help

.PHONY: release-snapshot
release-snapshot:              ## Build a snapshot release locally via goreleaser.
	@command -v goreleaser >/dev/null || { echo "install goreleaser: https://goreleaser.com/install/"; exit 1; }
	goreleaser release --snapshot --clean

.PHONY: clean
clean:                         ## Remove build artifacts.
	rm -rf $(BIN_DIR) dist/ cover.out cover.html

.PHONY: ci
ci: lint test                  ## What CI runs.
