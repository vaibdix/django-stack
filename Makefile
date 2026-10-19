.PHONY: build clean test test-race test-e2e test-generate test-docker cover cover-html lint fmt fmt-check vet arch goldens goldens-check ci ci-act install dev release-snapshot print-lint-version vulncheck help

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

test-race: ## Run the test suite under the race detector, recording coverage
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic ./...

test-e2e: ## Build the binary and drive it as a subprocess
	go test ./test/...

test-generate: ## Generate a real project from an answers file and run its own checks (network + uv)
	DJANGO_STACK_E2E=1 go test -count=1 -run TestAnswersFileGenerates -v ./test/e2e

test-docker: ## Start the generated compose files in real containers (requires docker)
	DJANGO_STACK_DOCKER_TESTS=1 go test -count=1 -run Live\|Healthy ./internal/scaffold/dbdocker

# The packages that own golden files and define the -update flag. Listed
# explicitly rather than as ./internal/... — passing -update to a package that
# does not define it fails with "flag provided but not defined", which is how
# this check silently did nothing until act ran it in a container.
GOLDEN_PKGS = ./internal/scaffold ./internal/assets ./internal/scaffold/dbdocker ./internal/tui

goldens: ## Regenerate the golden files after an intended output change
	go test $(GOLDEN_PKGS) -run Golden -update

goldens-check: ## Fail if the output no longer matches the goldens, or a golden is not committed
	@# Read-only: the golden tests compare without -update, and fail on a
	@# mismatch or a missing file. What they cannot see is a golden that exists
	@# but was never committed, which a fresh clone would not have.
	@go test -count=1 $(GOLDEN_PKGS) -run Golden
	@untracked=$$(git ls-files --others --exclude-standard -- '*testdata/golden*'); \
	if [ -n "$$untracked" ]; then \
		echo "golden files exist that are not committed:"; echo "$$untracked"; \
		echo "Review them, then commit. Regenerate with 'make goldens'."; \
		exit 1; \
	fi
	@echo "goldens up to date"

arch: ## Verify the scaffolding engine has no terminal dependencies
	@leaked=$$(go list -deps ./internal/scaffold/... ./internal/config/... ./internal/assets/... ./internal/pypi/... | grep charmbracelet || true); \
	if [ -n "$$leaked" ]; then \
		echo "a non-TUI package depends on Bubble Tea:"; echo "$$leaked"; exit 1; \
	fi; \
	echo "engine is free of terminal dependencies"

release-snapshot: ## Build release artefacts locally without publishing
	goreleaser release --snapshot --clean

cover: ## Run tests with coverage and print the total
	go test -coverprofile=$(COVERAGE) -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE) | tail -1

cover-html: cover ## Open the HTML coverage report
	go tool cover -html=$(COVERAGE)

fmt: ## Format all Go source
	gofmt -w ./cmd ./internal ./test

fmt-check: ## Fail if any file is not gofmt-clean
	@unformatted=$$(gofmt -l ./cmd ./internal ./test); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

vet: ## Run go vet
	go vet ./...

# The one place the linter version is pinned. CI installs this exact release
# and runs this target, so a local run and a CI run apply the same rules.
GOLANGCI_LINT_VERSION = 2.14.0

lint: ## Run golangci-lint at the pinned version
	@installed=$$(golangci-lint version --short 2>/dev/null); \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required, found '$${installed:-none}'."; \
		echo "Install it: https://golangci-lint.run/welcome/install/ (brew install golangci-lint, or the install script with v$(GOLANGCI_LINT_VERSION))"; \
		exit 1; \
	fi
	golangci-lint run --timeout=5m

# Pinned for the same reason as the linter: a new release must not change what
# a run reports without a commit that says so.
GOVULNCHECK_VERSION = v1.9.0

vulncheck: ## Fail if a reachable or required module has a known vulnerability
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) -show verbose ./...

ci: fmt-check vet arch goldens-check test-race test-e2e lint vulncheck ## Run everything CI runs

ci-act: ## Run the real CI workflow locally in containers (requires nektos/act + docker)
	@command -v act >/dev/null 2>&1 || { \
		echo "act is not installed: brew install act (see https://github.com/nektos/act)"; exit 1; }
	@# CI runs here with act, never on GitHub; .actrc points act at ci/workflows/ci.yml.
	@# DOCKER_HOST is set for OrbStack, which act cannot discover on its own.
	DOCKER_HOST="$${DOCKER_HOST:-unix://$$HOME/.orbstack/run/docker.sock}" \
		act push

install: ## Build and install into GOBIN
	@mkdir -p $(GOBIN_DIR)
	go build $(LDFLAGS) -o $(GOBIN_DIR)/$(APP_NAME) $(CLI_PKG)

dev: ## Hot-reload build (requires air: go install github.com/air-verse/air@latest)
	air

print-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)
