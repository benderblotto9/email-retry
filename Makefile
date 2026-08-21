BINARY   := email-retry
GO       ?= /usr/local/go/bin/go
MODULE   := $(shell $(GO) list -m)
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: all build clean test test-verbose test-race test-cover lint vet fmt check run help install

## ── Build ──────────────────────────────────────────────

all: check build ## Run checks then build

build: ## Build the binary
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: ## Install to $GOPATH/bin
	CGO_ENABLED=1 $(GO) install -ldflags "$(LDFLAGS)" .

run: ## Build and run (pass ARGS for flags)
	CGO_ENABLED=1 $(GO) run . $(ARGS)

clean: ## Remove build artifacts
	rm -f $(BINARY)
	$(GO) clean -cache

## ── Test ───────────────────────────────────────────────

test: ## Run tests
	CGO_ENABLED=1 $(GO) test ./...

test-verbose: ## Run tests with verbose output
	CGO_ENABLED=1 $(GO) test -v ./...

test-race: ## Run tests with race detector
	CGO_ENABLED=1 $(GO) test -race ./...

test-cover: ## Run tests with coverage report
	CGO_ENABLED=1 $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@rm -f coverage.out

test-cover-html: ## Generate HTML coverage report
	CGO_ENABLED=1 $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "coverage.html written"
	@rm -f coverage.out

## ── Code Quality ───────────────────────────────────────

check: fmt vet lint ## Run fmt + vet + lint

fmt: ## Format source files
	gofmt -s -w .

vet: ## Run go vet
	CGO_ENABLED=1 $(GO) vet ./...

lint: ## Run golangci-lint (if installed)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "⚠  golangci-lint not installed, skipping"; exit 0; }
	golangci-lint run ./...

## ── DB ─────────────────────────────────────────────────

clean-db: ## Remove the state database
	rm -f sent_emails.db*

## ── Help ───────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
