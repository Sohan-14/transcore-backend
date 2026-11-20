# ====================================================================================
# go-starter - developer commands.  Run `make` or `make help` to list them.
# Targets with a `## description` comment show up in help automatically.
# ====================================================================================

.DEFAULT_GOAL := help

# Load .env into the shell
WITH_ENV := set -a; if [ -f .env ]; then . ./.env; fi; set +a;

.PHONY: help

# .SILENT: don't show the command
# in the terminal when use make commands
.SILENT:

# ====================================================================================
##@ Everything
# ====================================================================================
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)


# ====================================================================================
##@ Testing & Quality
# ====================================================================================
test: ## Run all unit tests
	go test ./...

test-race: ## Run all tests with the race detector enabled
	go test -race ./...

test-cover: ## Run tests with coverage and print per-function summary
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

lint: ## Run golangci-lint using .golangci.yml
	golangci-lint run

check: lint test-race ## Run linter and race-enabled tests
