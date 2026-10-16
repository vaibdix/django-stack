.PHONY: build clean test test-race test-e2e test-docker refresh-django cover cover-html lint fmt fmt-check vet arch goldens goldens-check ci ci-act install dev release-snapshot help

APP_NAME   = django-stack
VERSION    = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     = $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
CLI_PKG    = ./cmd/django-stack
GOBIN_DIR  = $(shell if [ -n "$$(go env GOBIN)" ]; then go env GOBIN; else printf "%s/bin" "$$(go env GOPATH)"; fi)
LDVARS     = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)
LDFLAGS    = -ldflags "$(LDVARS)"
DIST       = dist
COVERAGE   = coverage.out

help: ## Show available targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## Build for the current platform into dist/
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(APP_NAME)-$(shell go env GOOS)-$(shell go env GOARCH) $(CLI_PKG)

clean: ## Remove build and coverage artifacts
	rm -rf $(DIST) $(COVERAGE)

test: ## Run the test suite
	go test ./...

test-race: ## Run the test suite under the race detector
	go test -race ./...

test-e2e: ## Build the binary and drive it as a subprocess
	go test ./test/...

test-docker: ## Start the generated compose files in real containers (requires docker)
	DJANGO_STACK_DOCKER_TESTS=1 go test -count=1 -run Live\|Healthy ./internal/scaffold/dbdocker

# The packages that own golden files and define the -update flag. Listed
# explicitly rather than as ./internal/... — passing -update to a package that
# does not define it fails with "flag provided but not defined", which is how
# this check silently did nothing until act ran it in a container.
GOLDEN_PKGS = ./internal/scaffold ./internal/assets ./internal/scaffold/dbdocker ./internal/tui

goldens: ## Regenerate the golden files after an intended output change
	go test $(GOLDEN_PKGS) -run Golden -update

goldens-check: ## Fail if regenerating the goldens would change anything
	@go test $(GOLDEN_PKGS) -run Golden -update
	@# `git status --porcelain`, not `git diff`: diff ignores untracked files, so
	@# on a tree where the goldens have never been committed this check would
	@# pass without comparing anything.
	@changed=$$(git status --porcelain -- '*testdata/golden*'); \
	if [ -n "$$changed" ]; then \
		echo "$$changed" | grep -q '^ *[MD]' && \
			echo "generated output changed and the golden files no longer match:" || \
			echo "golden files exist that are not committed:"; \
		echo "$$changed"; \
		echo "Review the diff, then commit. Regenerate with 'make goldens'."; \
		exit 1; \
	fi
	@echo "goldens up to date"

arch: ## Verify the scaffolding engine has no terminal dependencies
	@leaked=$$(go list -deps ./internal/scaffold/... ./internal/config/... ./internal/assets/... ./internal/pypi/... | grep charmbracelet || true); \
	if [ -n "$$leaked" ]; then \
		echo "a non-TUI package depends on Bubble Tea:"; echo "$$leaked"; exit 1; \
	fi; \
	echo "engine is free of terminal dependencies"

refresh-django: ## Refresh the Django releases bundled for offline use from PyPI
	go run ./internal/tools/djangoreleases -o internal/tui/django_fallback.go

release-snapshot: ## Build release artefacts locally without publishing
	goreleaser release --snapshot --clean

cover: ## Run tests with coverage and print the total
	go test -coverprofile=$(COVERAGE) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE) | tail -1

cover-html: cover ## Open the HTML coverage report
	go tool cover -html=$(COVERAGE)

fmt: ## Format all Go source
	gofmt -w ./cmd ./internal

fmt-check: ## Fail if any file is not gofmt-clean
	@unformatted=$$(gofmt -l ./cmd ./internal); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires golangci-lint)
	golangci-lint run

ci: fmt-check vet arch goldens-check test-race test-e2e lint ## Run everything CI runs

ci-act: ## Run the real CI workflow locally in containers (requires nektos/act + docker)
	@command -v act >/dev/null 2>&1 || { \
		echo "act is not installed: brew install act (see https://github.com/nektos/act)"; exit 1; }
	@# The workflows live in ci/workflows so they are versioned without running on GitHub.
	@# -W limits the run to ci.yml; release.yml fails locally on "no tags".
	@# DOCKER_HOST is set for OrbStack, which act cannot discover on its own.
	DOCKER_HOST="$${DOCKER_HOST:-unix://$$HOME/.orbstack/run/docker.sock}" \
		act push -W ci/workflows/ci.yml

install: ## Build and install into GOBIN
	@mkdir -p $(GOBIN_DIR)
	go build $(LDFLAGS) -o $(GOBIN_DIR)/$(APP_NAME) $(CLI_PKG)

dev: ## Hot-reload build (requires air: go install github.com/air-verse/air@latest)
	air
