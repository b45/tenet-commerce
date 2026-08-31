# Makefile for Tenet Commerce Backend
# Usage: make <target>

# --- Variables ---
BINARY_NAME=api
BUILD_DIR=./build
CMD_DIR=./cmd/api
PKG=./...

# --- Formatting ---
.PHONY: fmt
fmt: ## Format all Go source files
	gofmt -w .

# --- Linting ---
.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run $(PKG)

# --- Testing ---
.PHONY: test
test: ## Run all unit and integration tests with coverage
	go test -v -race -coverprofile=coverage.out $(PKG)

.PHONY: test-short
test-short: ## Run only unit tests (skips integration tests requiring DB)
	go test -v -short $(PKG)

.PHONY: coverage
coverage: test ## Generate HTML coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# --- Building ---
.PHONY: build
build: ## Build the API binary
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# --- Running ---
.PHONY: run
run: ## Run the API server (development mode)
	APP_DEBUG=true go run $(CMD_DIR)

# --- Docker ---
.PHONY: db-up
db-up: ## Start local PostgreSQL & Redis via Docker Compose
	docker compose -f ../docker-compose.yml up -d postgres redis

.PHONY: db-down
db-down: ## Stop local infrastructure
	docker compose -f ../docker-compose.yml down

.PHONY: db-reset
db-reset: ## Reset database and re-apply seed script
	docker compose -f ../docker-compose.yml down -v
	docker compose -f ../docker-compose.yml up -d postgres
	@echo "Waiting for postgres to be ready..."
	@sleep 3
	docker compose -f ../docker-compose.yml exec -T postgres psql -U postgres -d tenet_commerce -f /docker-entrypoint-initdb.d/01_init.sql

# --- Maintenance ---
.PHONY: tidy
tidy: ## Tidy and verify go modules
	go mod tidy
	go mod verify

.PHONY: vet
vet: ## Run go vet
	go vet $(PKG)

.PHONY: check
check: vet test ## Run vet + full test suite

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
