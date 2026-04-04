.PHONY: build build-all clean test install

APP_NAME = django-stack
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
CLI_PKG = ./cmd/cli
GOBIN_DIR = $(shell if [ -n "$$(go env GOBIN)" ]; then go env GOBIN; else printf "%s/bin" "$$(go env GOPATH)"; fi)
LDVARS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)
LDFLAGS = -ldflags "$(LDVARS)"
DIST = dist

# Build for current platform
build:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-$(shell go env GOOS)-$(shell go env GOARCH) $(CLI_PKG)

# Build for all platforms
build-all: clean
	@echo "Building $(APP_NAME) v$(VERSION) for multiple platforms..."
	@mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-windows-amd64.exe $(CLI_PKG)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-linux-amd64 $(CLI_PKG)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-linux-arm64 $(CLI_PKG)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-darwin-amd64 $(CLI_PKG)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-darwin-arm64 $(CLI_PKG)
	@echo "All builds completed!"
	@ls -la $(DIST)/

# Clean build artifacts
clean:
	rm -rf $(DIST)

# Run tests
test:
	go test -v ./...

# Install for current platform
install:
	@mkdir -p $(GOBIN_DIR)
	go build $(LDFLAGS) -o $(GOBIN_DIR)/$(APP_NAME) $(CLI_PKG)

# Development build with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Lint code (requires golangci-lint)
lint:
	golangci-lint run
