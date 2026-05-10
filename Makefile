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

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race ./...

.PHONY: lint
lint: ## Run linters
	gofmt -d -s .
	go vet ./...

.PHONY: fmt
fmt: ## Format the code
	gofmt -s -w .
	go mod tidy

## TODO: add coverage ++ get to 100%
