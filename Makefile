# ==============================================================================
# Tenet Commerce - Master Workspace Makefile
# ==============================================================================

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Infrastructure & Docker ---
.PHONY: db-up
db-up: ## Start local PostgreSQL & Redis via Docker Compose
	docker compose up -d postgres redis

.PHONY: db-down
db-down: ## Stop local infrastructure containers
	docker compose down

.PHONY: db-reset
db-reset: ## Reset local PostgreSQL database and re-apply seed data (opt-in demo guard)
	@CONFIRM_DEMO_RESET=$${CONFIRM_DEMO_RESET:-true} ./scripts/reset_dev_db.sh


# --- Backend Commands ---
.PHONY: run
run: ## Run the backend Go API server in development mode
	cd backend && APP_DEBUG=true go run ./cmd/api

.PHONY: test
test: ## Run backend tests, including the Testcontainers integration suite
	cd backend && go test -v -race ./...

.PHONY: test-integration
test-integration: ## Run hermetic PostgreSQL/Redis integration tests only
	cd backend && go test -v -race ./integration

.PHONY: build
build: ## Build backend production binary
	cd backend && go build -ldflags="-s -w" -o build/api ./cmd/api

.PHONY: tidy
tidy: ## Tidy backend go modules
	cd backend && go mod tidy

# --- Frontend Commands ---
.PHONY: fe-dev
fe-dev: ## Run frontend Next.js dev server with live logs (default: port 3000, or `make fe-dev PORT=3001`)
	cd frontend && npm run dev -- -p $(or $(PORT),3000)

.PHONY: fe-clean
fe-clean: ## Clean Next.js build cache (.next)
	rm -rf frontend/.next

.PHONY: fe-test
fe-test: ## Run frontend unit and i18n parity test suites
	cd frontend && npm test

.PHONY: fe-lint
fe-lint: ## Run frontend ESLint and TypeScript checks
	cd frontend && npm run lint && npx tsc --noEmit

.PHONY: fe-build
fe-build: ## Build frontend production bundle
	cd frontend && npm run build

.DEFAULT_GOAL := help
