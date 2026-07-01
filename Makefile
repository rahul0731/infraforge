.PHONY: help up down build run migrate logs clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Docker
up: ## Start all services (PostgreSQL, Temporal, Temporal UI)
	docker compose up -d

down: ## Stop all services
	docker compose down

down-clean: ## Stop all services and remove volumes
	docker compose down -v

logs: ## Tail service logs
	docker compose logs -f

logs-temporal: ## Tail Temporal logs
	docker compose logs -f temporal

# Backend
build: ## Build the Go backend
	cd backend && go build -o ../bin/infraforge-server ./cmd/server
	cd backend && go build -o ../bin/infraforge-worker ./cmd/worker

run: ## Run the Go API server locally
	cd backend && go run ./cmd/server

worker: ## Run the Temporal worker locally
	cd backend && go run ./cmd/worker

tidy: ## Tidy Go modules
	cd backend && go mod tidy

# Database
migrate: ## Run database migrations (applies SQL to running Postgres)
	@echo "Applying migrations to PostgreSQL..."
	docker compose exec postgres psql -U infraforge -d infraforge -f /docker-entrypoint-initdb.d/001_initial_schema.sql

psql: ## Open psql shell
	docker compose exec postgres psql -U infraforge -d infraforge

# Clean
clean: ## Remove build artifacts
	rm -rf bin/
