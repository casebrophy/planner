# ==============================================================================
# Planner

-include .env
export

COMPOSE := docker compose -f zarf/compose/docker-compose.yml

# ==============================================================================
# Development

dev:
	go run api/services/planner/main.go

admin:
	go run api/tooling/admin/main.go $(ARGS)

migrate:
	go run api/tooling/admin/main.go migrate

seed:
	go run api/tooling/admin/main.go seed

# ==============================================================================
# Docker

build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart backend

logs:
	$(COMPOSE) logs -f backend

db-up:
	$(COMPOSE) up -d db

db-down:
	$(COMPOSE) down db

# ==============================================================================
# Testing and Linting

test:
	go test ./... -count=1

lint:
	go vet ./...

tidy:
	go mod tidy

# ==============================================================================
# Docker Migrate/Seed

docker-migrate:
	$(COMPOSE) exec backend /service/admin migrate

docker-seed:
	$(COMPOSE) exec backend /service/admin seed

# ==============================================================================
# Help

help:
	@echo "Usage:"
	@echo "  make dev            - Run the API locally"
	@echo "  make admin ARGS=cmd - Run the admin tool"
	@echo "  make migrate        - Run database migrations (local)"
	@echo "  make seed           - Seed the database (local)"
	@echo "  make build          - Build Docker images"
	@echo "  make up             - Start all containers"
	@echo "  make down           - Stop all containers"
	@echo "  make db-up          - Start just the database"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linter"
	@echo "  make tidy           - Run go mod tidy"
