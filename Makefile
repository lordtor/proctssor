.PHONY: help up down build test test-unit test-integration test-load lint clean

# Default values
COMPOSE_FILE := docker-compose.yaml
PROJECT_NAME := workflow-engine

# Colors for output
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "Workflow Engine - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

up: ## Start all services
	@echo "$(GREEN)Starting all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d
	sleep 5
	@make wait-for-services

down: ## Stop all services
	@echo "$(RED)Stopping all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down

build: ## Build all Docker images
	@echo "$(GREEN)Building all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) build

restart: down up ## Restart all services

wait-for-services: ## Wait for services to be ready
	@echo "$(YELLOW)Waiting for services to be ready...$(NC)"
	@./scripts/wait-for-service.sh postgres 5432 30
	@./scripts/wait-for-service.sh nats 4222 30
	@echo "$(GREEN)All services are ready!$(NC)"

# Testing commands
test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "$(GREEN)Running unit tests...$(NC)"
	cd engine && go test -v ./internal/core/... -coverprofile=coverage-core.out
	cd engine && go test -v ./internal/service/... -coverprofile=coverage-service.out

test-integration: ## Run integration tests with testcontainers
	@echo "$(GREEN)Running integration tests...$(NC)"
	cd engine && go test -v ./tests/integration/... -tags=integration -timeout=20m

test-bpmn: ## Run BPMN parser tests
	@echo "$(GREEN)Running BPMN parser tests...$(NC)"
	cd engine && go test -v ./internal/core/bpmn/... -coverprofile=coverage-bpmn.out

test-coverage: ## Generate test coverage report
	@echo "$(GREEN)Generating test coverage report...$(NC)"
	cd engine && go test -coverprofile=coverage.out ./...
	cd engine && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: engine/coverage.html$(NC)"

# Load testing commands


# CI commands
ci-test: ## Run tests in CI environment
	@echo "$(GREEN)Running CI tests...$(NC)"
	cd engine && go test -v ./internal/core/bpmn/... -race -coverprofile=coverage.out
	cd engine && go test -v ./internal/core/statemachine/... -race
	cd engine && go test -v ./internal/core/saga/... -race
	cd engine && go test -v ./internal/service/... -race

ci-lint: ## Run linters
	@echo "$(GREEN)Running linters...$(NC)"
	cd engine && golangci-lint run ./...

ci-build: ## Build for CI
	@echo "$(GREEN)Building for CI...$(NC)"
	docker build -t workflow-engine:latest -f engine/Dockerfile engine/

# Development commands
dev-logs: ## Show service logs
	docker-compose -f $(COMPOSE_FILE) logs -f engine

dev-logs-all: ## Show all service logs
	docker-compose -f $(COMPOSE_FILE) logs -f

dev-shell: ## Open shell in engine container
	docker-compose -f $(COMPOSE_FILE) exec engine /bin/sh

# Database commands
db-migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	docker-compose -f $(COMPOSE_FILE) exec postgres psql -U workflow -d workflow -f /docker-entrypoint-initdb.d/01_schemas.sql

db-seed: ## Seed database with test data
	@echo "$(GREEN)Seeding database...$(NC)"
	@echo "$(YELLOW)Not implemented yet$(NC)"

db-reset: ## Reset database (drop and recreate)
	@echo "$(RED)Resetting database...$(NC)"
	docker-compose -f $(COMPOSE_FILE) exec postgres psql -U workflow -d workflow -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@make db-migrate

# Registry commands
registry-list: ## List registered services
	@echo "$(GREEN)Listing registered services...$(NC)"
	@curl -s http://localhost:8080/api/v1/registry/services | jq . || echo "Failed to list services"

# Cleanup commands
clean: down ## Clean up all containers and volumes
	@echo "$(RED)Cleaning up containers and volumes...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v --remove-orphans
	docker system prune -f

clean-images: ## Remove all built images
	@echo "$(RED)Removing Docker images...$(NC)"
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down --rmi all

# Dependency management
deps: ## Download Go dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	cd engine && go mod download
	cd engine && go mod tidy

deps-update: ## Update Go dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	cd engine && go get -u ./...
	cd engine && go mod tidy

# Code quality
fmt: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	cd engine && gofmt -w ./
	cd engine && goimports -w ./

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	cd engine && go vet ./...

generate: ## Generate code (mocks, etc.)
	@echo "$(GREEN)Generating code...$(NC)"
	cd engine && go generate ./...

# Documentation
docs: ## Generate documentation
	@echo "$(GREEN)Generating documentation...$(NC)"
	cd engine && swag init -g cmd/engine/main.go -o docs/

# Deployment
deploy-local: build up ## Deploy locally
	@echo "$(GREEN)Local deployment complete$(NC)"

# E2E tests (requires running services)
test-e2e: ## Run E2E tests (requires make up)
	@echo "$(YELLOW)E2E tests not implemented yet$(NC)"

# Full CI pipeline
ci: ci-lint ci-test ci-build ## Run full CI pipeline
	@echo "$(GREEN)CI pipeline complete!$(NC)"
