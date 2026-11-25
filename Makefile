# ====================================================================================
# go-starter - developer commands.  Run `make` or `make help` to list them.
# Targets with a `## description` comment show up in help automatically.
# ====================================================================================

MIGRATIONS_DIR = ./migrations
BIN_DIR  := ./bin

.DEFAULT_GOAL := help

# Load .env into the shell
WITH_ENV := set -a; if [ -f .env ]; then . ./.env; fi; set +a;

.PHONY: help test test-race test-cover lint check migrate-new migrate-up migrate-down migrate-status up down clean sqlc

# .SILENT: don't show the command
# in the terminal when use make commands
# .SILENT:

# ====================================================================================
##@ Everything
# ====================================================================================
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)


# ====================================================================================
##@ Run and build
# ====================================================================================
run: ## Run the API locally (loads .env)
	@$(WITH_ENV) go run  ./cmd/api

build: ## Build binaries into ./bin
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -o $(BIN_DIR)/api ./cmd/api

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

# ====================================================================================
##@ Database & Migration
# ====================================================================================
migrate-new: ## Create a new SQL migration (usage: make migrate-new name=create_accounts)
	@test -n "$(name)" || (echo "name is required: make migrate-new name=..." && exit 1)
	@$(WITH_ENV) goose -s -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-up: ## Run migrations up
	@$(WITH_ENV) goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" up

migrate-down: ## Rollback the last migration
	@$(WITH_ENV) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status: ## See migration status
	@$(WITH_ENV) goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

# ====================================================================================
##@ Docker
# ====================================================================================
up: ## Start docker (docker compose)
	$(WITH_ENV) docker compose up -d --wait

down: ## Stop docker (data is kept; add -v to the compose command to wipe it)
	$(WITH_ENV) docker compose down

clean: ## Stop & clean docker
	$(WITH_ENV) docker compose down -v


# ====================================================================================
##@ Sqlc
# ====================================================================================
sqlc: ## generate type-safe queries
	@sqlc generate
