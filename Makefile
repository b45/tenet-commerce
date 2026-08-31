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
db-reset: ## Reset local PostgreSQL database and re-apply seed data
	docker compose down -v
	docker compose up -d postgres redis
	@echo "Waiting for postgres to be ready..."
	@sleep 3
	docker exec -i tenet_postgres psql -U postgres -d tenet_commerce < scripts/init_dev_db.sql
	@echo "Database reset and seeded successfully!"

# --- Backend Commands ---
.PHONY: run
run: ## Run the backend Go API server in development mode
	cd backend && APP_DEBUG=true go run ./cmd/api

.PHONY: test
test: ## Run backend unit and integration tests
	cd backend && go test -v -race ./...

.PHONY: build
build: ## Build backend production binary
	cd backend && go build -ldflags="-s -w" -o build/api ./cmd/api

.PHONY: tidy
tidy: ## Tidy backend go modules
	cd backend && go mod tidy

.DEFAULT_GOAL := help
