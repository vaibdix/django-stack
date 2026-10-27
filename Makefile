.PHONY: test-live content-check repo-clean comments cross deadcode build clean test test-short test-race test-e2e test-generate test-docker cover cover-html lint fmt fmt-check vet arch goldens goldens-check ci ci-act outdated install release-snapshot print-lint-version vulncheck help

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

test-short: ## Run the suite without the tests that download packages (works offline)
	go test -short ./...

test-race: ## Run the test suite under the race detector, recording coverage
	go test -race -shuffle=on -count=3 -coverprofile=$(COVERAGE) -covermode=atomic ./...

test-e2e: ## Build the binary and drive it as a subprocess
	go test ./test/...

test-generate: ## Generate a real project from an answers file and run its own checks (network + uv)
	DJANGO_STACK_LIVE=1 DJANGO_STACK_E2E=1 go test -count=1 -run TestAnswersFileGenerates -v ./test/e2e

test-docker: ## Start the generated compose files in real containers (requires docker)
	DJANGO_STACK_LIVE=1 go test -count=1 -run Live\|Healthy ./internal/dbdocker

test-live: ## Run every test, including the network, Docker and end-to-end ones (DJANGO_STACK_LIVE=1)
	DJANGO_STACK_LIVE=1 DJANGO_STACK_E2E=1 go test -count=1 ./...

content-check: ## Render every configuration and run py_compile, ruff, djlint and docker compose config on each unique file
	@dir=$$(mktemp -d); go run ./tools/matrix render-all -run "$$dir"; status=$$?; rm -rf "$$dir"; exit $$status

GOLDEN_PKGS = ./internal/render ./internal/assets ./internal/dbdocker ./internal/tui

goldens: ## Regenerate the golden files after an intended output change
	go test $(GOLDEN_PKGS) -run Golden -update-goldens

goldens-check: ## Fail if the output no longer matches the goldens, or a golden is not committed
	@go test -count=1 $(GOLDEN_PKGS) -run Golden
	@untracked=$$(git ls-files --others --exclude-standard -- '*testdata/golden*'); \
	if [ -n "$$untracked" ]; then \
		echo "golden files exist that are not committed:"; echo "$$untracked"; \
		echo "Review them, then commit. Regenerate with 'make goldens'."; \
		exit 1; \
	fi
	@echo "goldens up to date"

arch: ## Verify the package layering rules (depguard)
	golangci-lint run --enable-only depguard

release-snapshot: ## Build release artefacts, SBOMs and the signed checksums locally without publishing
	@for tool in goreleaser syft cosign; do command -v $$tool >/dev/null 2>&1 || { \
		echo "$$tool is required: brew install goreleaser syft cosign"; exit 1; }; done
	goreleaser release --snapshot --clean

COVDATA = $(DIST)/covdata

cover: ## Run the unit and end-to-end tests with coverage across every package and print the total
	@rm -rf $(COVDATA) && mkdir -p $(COVDATA)/unit $(COVDATA)/e2e
	go test -cover -coverpkg=./... $$(go list ./... | grep -v /test/) -args -test.gocoverdir=$(CURDIR)/$(COVDATA)/unit
	go test -c -o $(DIST)/e2e.test ./test/e2e
	cd test/e2e && GOFLAGS=-cover GOCOVERDIR=$(CURDIR)/$(COVDATA)/e2e $(CURDIR)/$(DIST)/e2e.test
	go tool covdata textfmt -i=$(COVDATA)/unit,$(COVDATA)/e2e -o $(COVERAGE)
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

DEADCODE_TEST_SUPPORT = render.go:.*(TemplateNames|TemplateFS|Unrendered)|WriteComposeFile|Nop[.]

deadcode: ## Fail on any function no binary reaches on any OS (exported test support excepted)
	@go build -o $(DIST)/deadcode golang.org/x/tools/cmd/deadcode
	@for os in $(CROSS_OS); do GOOS=$$os $(DIST)/deadcode ./cmd/... ./tools/... | sort > $(DIST)/deadcode-$$os.txt || exit 1; done
	@dead=$$(comm -12 $(DIST)/deadcode-windows.txt $(DIST)/deadcode-darwin.txt | comm -12 - $(DIST)/deadcode-linux.txt | grep -vE '$(DEADCODE_TEST_SUPPORT)' || true); \
	if [ -n "$$dead" ]; then echo "unreachable on every OS:"; echo "$$dead"; exit 1; fi; \
	echo "no function is unreachable on every OS"

CROSS_OS = windows darwin linux

cross: ## Vet and compile every package and test for Windows, macOS and Linux
	@for os in $(CROSS_OS); do \
		echo "GOOS=$$os"; \
		GOOS=$$os go vet ./... || exit 1; \
		for pkg in $$(go list ./...); do GOOS=$$os go test -c -o /dev/null $$pkg || exit 1; done; \
	done

GOLANGCI_LINT_VERSION = 2.15.0

lint: ## Run golangci-lint at the pinned version
	@installed=$$(golangci-lint version --short 2>/dev/null); \
	if [ "$$installed" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) is required, found '$${installed:-none}'."; \
		echo "Install it: https://golangci-lint.run/welcome/install/ (brew install golangci-lint, or the install script with v$(GOLANGCI_LINT_VERSION))"; \
		exit 1; \
	fi
	golangci-lint run --timeout=5m


vulncheck: ## Fail if a reachable or required module has a known vulnerability
	go tool govulncheck -show verbose ./...

OUTDATED_HOLDS =

outdated: ## Report pins, actions, images and modules that are behind their latest release
	go run ./tools/outdated $(foreach hold,$(OUTDATED_HOLDS),-hold $(hold))
	DJANGO_STACK_LIVE=1 go test -count=1 -run 'TestThePinned' ./internal/render

repo-clean: ## Fail if run output or agent notes are tracked
	@tracked=$$(git ls-files --cached --ignored --exclude-standard); \
	if [ -n "$$tracked" ]; then \
		echo "tracked files match .gitignore:"; echo "$$tracked"; exit 1; \
	fi; \
	echo "no ignored file is tracked"

SCC_VERSION = v3.7.0
NOCOMMENTS  = $(CURDIR)/$(DIST)/nocomments

comments: ## Fail if Go code or repo config carries a comment (directives and ## help are allowed)
	@go build -o $(NOCOMMENTS) ./tools/nocomments
	@go vet -vettool=$(NOCOMMENTS) $$(go list ./... | grep -v /test/)
	@commented=$$(go run github.com/boyter/scc/v3@$(SCC_VERSION) --no-cocomo --by-file -f csv .golangci.yml .goreleaser.yaml ci Makefile | awk -F, 'NR > 1 && $$6 > 0 {print $$2}'); \
	if [ -n "$$commented" ]; then echo "comments in repo config:"; echo "$$commented"; exit 1; fi; \
	echo "no comments in Go or repo config"

ci: repo-clean comments fmt-check vet cross deadcode arch goldens-check test-race test-e2e lint vulncheck ## Run everything CI runs

ci-act: ## Run the real CI workflow locally in containers (requires nektos/act + docker)
	@command -v act >/dev/null 2>&1 || { \
		echo "act is not installed: brew install act (see https://github.com/nektos/act)"; exit 1; }
	DOCKER_HOST="$${DOCKER_HOST:-unix://$$HOME/.orbstack/run/docker.sock}" \
		act push

install: ## Build and install into GOBIN
	@mkdir -p $(GOBIN_DIR)
	go build $(LDFLAGS) -o $(GOBIN_DIR)/$(APP_NAME) $(CLI_PKG)

print-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)
