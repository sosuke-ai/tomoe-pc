.PHONY: dev-deps dev-tools fmt lint vet test test-integration test-coverage build build-cuda package install clean download-model

BINARY    := tomoe
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-X main.Version=$(VERSION)"
GOFLAGS   := -v
INSTALL_DIR := /usr/local/bin
LINT_VERSION := v1.64.8

## Development setup ──────────────────────────────────────────────────

dev-deps: ## Install Ubuntu system packages needed for development
	sudo apt install -y build-essential pkg-config \
	  libx11-dev libxtst-dev libxkbcommon-dev \
	  libasound-dev portaudio19-dev libportaudio2 libpulse-dev \
	  xclip xdotool wl-clipboard wtype libnotify-bin ffmpeg

dev-tools: ## Install Go development tools (golangci-lint, goimports)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINT_VERSION)
	go install golang.org/x/tools/cmd/goimports@latest

## Code quality ───────────────────────────────────────────────────────

fmt: ## Format Go source files
	gofmt -w .
	goimports -w .

lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

## Testing ────────────────────────────────────────────────────────────

test: ## Run unit tests
	go test $(GOFLAGS) ./...

test-integration: ## Run integration tests (requires model + hardware)
	go test $(GOFLAGS) -tags integration -timeout 120s ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Build ──────────────────────────────────────────────────────────────

build: ## Build CLI binary
	CGO_ENABLED=1 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/tomoe

build-cuda: build ## Same binary — CUDA EP is selected at runtime via config

## Distribution ───────────────────────────────────────────────────────

package: build ## Create release tarball
	mkdir -p dist
	tar czf dist/$(BINARY)-linux-amd64.tar.gz $(BINARY) README.md LICENSE docs/

## Install / Clean ────────────────────────────────────────────────────

install: build ## Install to /usr/local/bin
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist/ coverage.out coverage.html

## Model ──────────────────────────────────────────────────────────────

download-model: build ## Download Parakeet TDT INT8 model + Silero VAD
	./$(BINARY) model download

## Help ───────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
