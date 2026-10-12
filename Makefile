.DEFAULT_GOAL := help

VERSION := $(shell git describe --tags --always --dirty="-dev")

##@ Help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: v
v: ## Show the version
	@echo "Version: ${VERSION}"

##@ Build

.PHONY: clean
clean: ## Clean the build directory
	rm -rf build/

.PHONY: build
build: ## Build the linter
	@mkdir -p ./build
	go build -trimpath -ldflags "-X main.Version=${VERSION}" -v -o ./build/openrpc-linter .

##@ Test & Development

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: test-e2e
test-e2e: ## Run Godog e2e tests
	go test ./e2e

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race ./...

.PHONY: lint
lint: ## Run linters
	@test -z "$$(gofmt -l -s .)" || { gofmt -d -s .; exit 1; }
	go vet ./...
	# Keep golangci-lint v1 on its supported Go toolchain, including locally.
	GOTOOLCHAIN=go1.24.5 go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run --timeout=5m

.PHONY: fmt
fmt: ## Format the code
	gofmt -s -w .
	go mod tidy

.PHONY: quality-tool quality-gate quality-baseline
quality-tool: ## Install the pinned Gstyle version for the quality gate
	@mkdir -p .tools
	GOBIN="$(CURDIR)/.tools" GOTOOLCHAIN=go1.27.1 go install github.com/rstyled/gstyle@$$(bun -p 'JSON.parse(await Bun.file("quality-baseline.json").text()).gstyle_version')

quality-gate: quality-tool ## Check findings per 1000 production functions
	bun test ./.github/workflows/quality-gate.test.ts
	GSTYLE="$(CURDIR)/.tools/gstyle" bun .github/workflows/quality-gate.ts

quality-baseline: quality-tool ## Refresh the quality baseline for review
	GSTYLE="$(CURDIR)/.tools/gstyle" bun .github/workflows/quality-gate.ts --write
