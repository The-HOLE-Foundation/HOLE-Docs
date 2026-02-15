.PHONY: help build run test clean install install-local release release-macos release-linux clean-dist

# Variables
BIN_NAME=hole-docs
BIN_PATH=./bin/$(BIN_NAME)
GO=go
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST_DIR=./dist

help: ## Show this help message
	@echo "hole-docs - Free PDF Tools for Transparency Workflows"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build hole-docs binary
	$(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(BIN_PATH) ./cmd/hole-docs
	@echo "✅ Build complete: $(BIN_PATH)"
	@echo "   Test with: $(BIN_PATH) help"

run: build ## Build and run
	$(BIN_PATH) help

test: ## Run tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf $(DIST_DIR)/
	rm -f coverage.out coverage.html
	$(GO) clean

install: build ## Install to /usr/local/bin (requires sudo)
	mkdir -p /usr/local/bin
	cp $(BIN_PATH) /usr/local/bin/hole-docs
	@echo "✅ Installed to /usr/local/bin/hole-docs"
	@echo "   Test with: hole-docs help"

install-local: build ## Install to ~/.local/bin (no sudo)
	@mkdir -p $(HOME)/.local/bin
	cp $(BIN_PATH) $(HOME)/.local/bin/hole-docs
	@echo "✅ Installed to $(HOME)/.local/bin/hole-docs"
	@echo "   Add to PATH: export PATH=\"\$$HOME/.local/bin:\$$PATH\""

fmt: ## Format code
	$(GO) fmt ./...

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

vet: ## Run go vet
	$(GO) vet ./...

check: fmt vet test ## Run all checks

# Cross-compilation targets
release-macos-arm64: ## Build for macOS ARM64 (M1/M2/M3)
	@echo "Building for macOS ARM64..."
	@mkdir -p $(DIST_DIR)/macos-arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(DIST_DIR)/macos-arm64/hole-docs ./cmd/hole-docs
	@cd $(DIST_DIR)/macos-arm64 && tar -czf ../hole-docs-$(VERSION)-macos-arm64.tar.gz hole-docs
	@cd $(DIST_DIR) && shasum -a 256 hole-docs-$(VERSION)-macos-arm64.tar.gz > hole-docs-$(VERSION)-macos-arm64.tar.gz.sha256
	@echo "✅ macOS ARM64: $(DIST_DIR)/hole-docs-$(VERSION)-macos-arm64.tar.gz"

release-macos-amd64: ## Build for macOS Intel
	@echo "Building for macOS AMD64..."
	@mkdir -p $(DIST_DIR)/macos-amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(DIST_DIR)/macos-amd64/hole-docs ./cmd/hole-docs
	@cd $(DIST_DIR)/macos-amd64 && tar -czf ../hole-docs-$(VERSION)-macos-amd64.tar.gz hole-docs
	@cd $(DIST_DIR) && shasum -a 256 hole-docs-$(VERSION)-macos-amd64.tar.gz > hole-docs-$(VERSION)-macos-amd64.tar.gz.sha256
	@echo "✅ macOS AMD64: $(DIST_DIR)/hole-docs-$(VERSION)-macos-amd64.tar.gz"

release-linux-amd64: ## Build for Linux AMD64
	@echo "Building for Linux AMD64..."
	@mkdir -p $(DIST_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(DIST_DIR)/linux-amd64/hole-docs ./cmd/hole-docs
	@cd $(DIST_DIR)/linux-amd64 && tar -czf ../hole-docs-$(VERSION)-linux-amd64.tar.gz hole-docs
	@cd $(DIST_DIR) && sha256sum hole-docs-$(VERSION)-linux-amd64.tar.gz > hole-docs-$(VERSION)-linux-amd64.tar.gz.sha256
	@echo "✅ Linux AMD64: $(DIST_DIR)/hole-docs-$(VERSION)-linux-amd64.tar.gz"

release-linux-arm64: ## Build for Linux ARM64
	@echo "Building for Linux ARM64..."
	@mkdir -p $(DIST_DIR)/linux-arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags="-s -w -X main.Version=$(VERSION)" -o $(DIST_DIR)/linux-arm64/hole-docs ./cmd/hole-docs
	@cd $(DIST_DIR)/linux-arm64 && tar -czf ../hole-docs-$(VERSION)-linux-arm64.tar.gz hole-docs
	@cd $(DIST_DIR) && sha256sum hole-docs-$(VERSION)-linux-arm64.tar.gz > hole-docs-$(VERSION)-linux-arm64.tar.gz.sha256
	@echo "✅ Linux ARM64: $(DIST_DIR)/hole-docs-$(VERSION)-linux-arm64.tar.gz"

release-macos: release-macos-arm64 release-macos-amd64 ## Build all macOS releases

release-linux: release-linux-amd64 release-linux-arm64 ## Build all Linux releases

release: release-macos release-linux ## Build all platform releases
	@echo ""
	@echo "✅ All releases built!"
	@ls -lh $(DIST_DIR)/*.tar.gz
	@echo ""
	@echo "Checksums:"
	@cat $(DIST_DIR)/*.sha256
	@echo ""
	@echo "To create a GitHub release:"
	@echo "  git tag v$(VERSION)"
	@echo "  git push origin v$(VERSION)"

clean-dist: ## Clean distribution directory
	rm -rf $(DIST_DIR)

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

.DEFAULT_GOAL := help
