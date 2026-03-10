.PHONY: dev-deps dev-tools fmt lint vet test test-integration test-coverage build build-gui build-cuda package install clean download-model dev-gui

BINARY      := tomoe
GUI_BINARY  := tomoe-gui
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-X main.Version=$(VERSION)"
GOFLAGS     := -v
GOBIN       := $(shell go env GOPATH)/bin
INSTALL_DIR := $(GOBIN)
LINT_VERSION := v2.11.3

# Auto-detect webkit2gtk for GUI build
HAS_WEBKIT := $(shell pkg-config --exists webkit2gtk-4.1 2>/dev/null && echo yes || echo no)

## Development setup ──────────────────────────────────────────────────

dev-deps: ## Install Ubuntu system packages needed for development
	sudo apt install -y build-essential pkg-config \
	  libx11-dev libxtst-dev libxkbcommon-dev \
	  libasound-dev portaudio19-dev libportaudio2 libpulse-dev \
	  xclip xdotool wl-clipboard wtype libnotify-bin ffmpeg \
	  libwebkit2gtk-4.1-dev libappindicator3-dev libgtk-3-dev
	@echo "Installing Node.js dependencies for frontend..."
	cd frontend && npm install

dev-tools: ## Install Go development tools (golangci-lint, goimports, wails)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION)
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/wailsapp/wails/v2/cmd/wails@latest

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

build: ## Build CLI binary (and GUI if webkit2gtk available)
	CGO_ENABLED=1 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/tomoe
ifeq ($(HAS_WEBKIT),yes)
	@echo "webkit2gtk-4.1 detected, building GUI..."
	CGO_ENABLED=1 go build $(GOFLAGS) $(LDFLAGS) -tags production,webkit2_41 -o $(GUI_BINARY) ./cmd/tomoe-gui
else
	@echo "webkit2gtk-4.1 not found, skipping GUI build."
endif

build-gui: build-frontend ## Build GUI binary only (requires webkit2gtk-4.1)
	rm -rf cmd/tomoe-gui/frontend/dist
	cp -r frontend/dist cmd/tomoe-gui/frontend/dist
	CGO_ENABLED=1 go build $(GOFLAGS) $(LDFLAGS) -tags production,webkit2_41 -o $(GUI_BINARY) ./cmd/tomoe-gui

build-frontend: ## Build React frontend
	cd frontend && npm install && npm run build

build-cuda: build ## Same binary — CUDA EP is selected at runtime via config

dev-gui: ## Run Wails dev mode with hot-reload
	cd frontend && npm install
	wails dev

## Distribution ───────────────────────────────────────────────────────

package: build ## Create release tarball
	mkdir -p dist
	tar czf dist/$(BINARY)-linux-amd64.tar.gz $(BINARY) $(wildcard $(GUI_BINARY)) README.md LICENSE docs/

## Install / Clean ────────────────────────────────────────────────────

install: build ## Install to GOPATH/bin
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
ifeq ($(HAS_WEBKIT),yes)
	install -m 755 $(GUI_BINARY) $(INSTALL_DIR)/$(GUI_BINARY)
endif

clean: ## Remove build artifacts
	rm -f $(BINARY) $(GUI_BINARY)
	rm -rf dist/ coverage.out coverage.html

## Model ──────────────────────────────────────────────────────────────

download-model: build ## Download Parakeet TDT INT8 model + Silero VAD + Speaker Embedding
	./$(BINARY) model download

## Help ───────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
