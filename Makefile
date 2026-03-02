# BPMN Workflow Platform - Makefile
# Development commands for local development

.PHONY: help up down logs build clean install test setup reset

# Default target
help:
	@echo "BPMN Workflow Platform - Available commands:"
	@echo "  make setup     - Initialize project (copy env, pull images)"
	@echo "  make up       - Start all services"
	@echo "  make down     - Stop all services"
	@echo "  make logs     - Show logs from all services"
	@echo "  make build    - Build all Docker images"
	@echo "  make reset    - Stop and remove all (down -v)"
	@echo "  make install  - Install dependencies"
	@echo "  make test     - Run tests"

# Start all services
up:
	docker-compose up -d
	@echo "Services started. Use 'make logs' to view logs."

# Stop all services
down:
	docker-compose down

# Show logs from all services
logs:
	docker-compose logs -f

# Build all Docker images
build:
	docker-compose build

# Clean up containers and volumes
clean:
	docker-compose down -v
	@echo "Containers and volumes removed."

# Reset - stop and remove everything
reset:
	docker-compose down -v
	@echo "All services stopped and volumes removed."

# Install dependencies
install:
	@echo "Installing dependencies..."
	cd engine && go mod download
	cd web && npm install

# Setup - initialize project
setup:
	@echo "Setting up BPMN Workflow Platform..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@echo "Project initialized. Run 'make up' to start services."

# Run tests
test:
	@echo "Running tests..."
	cd engine && go test ./...
	cd web && npm test

# Development server - engine only
dev-engine:
	cd engine && go run main.go

# Development server - web only
dev-web:
	cd web && npm run dev

