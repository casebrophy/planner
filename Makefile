# ==============================================================================
# Planner

-include .env
-include .env.local
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

NPM := npm --prefix api/services/frontend/web

dev-up: db-up migrate
	@trap 'kill 0' EXIT; \
	go run api/services/planner/main.go & \
	$(NPM) run dev

dev-up-full: db-up migrate
	@trap 'kill 0' EXIT; \
	go run api/services/planner/main.go & \
	SIDECAR_LOG_PATH=/tmp/planner-sidecar-requests.jsonl go run zarf/sidecar/*.go & \
	$(NPM) run dev

dev-down: db-down

frontend-dev:
	$(NPM) run dev

frontend-build:
	$(NPM) run build

frontend-serve:
	$(NPM) run build && PLANNER_FRONTEND_DIR=api/services/frontend/web/dist go run api/services/frontend/main.go

frontend-test:
	$(NPM) test

frontend-lint:
	$(NPM) run lint

frontend-install:
	$(NPM) install

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

logs-all:
	$(COMPOSE) logs -f backend frontend

db-up:
	$(COMPOSE) up -d db

db-down:
	$(COMPOSE) down db

db:
	$(COMPOSE) exec db psql -U planner planner

ollama-pull:
	docker exec -it planner-ollama ollama pull llama3

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
# iOS (Capacitor)

cap-build:
	cd api/services/frontend/web && CAPACITOR_BUILD=true npm run build && npx cap sync ios

cap-open:
	cd api/services/frontend/web && npx cap open ios

# Pass-through: make npm ARGS="install axios"
npm:
	$(NPM) $(ARGS)

# ==============================================================================
# Deploy

deploy:
	./zarf/deploy/deploy.sh

# ==============================================================================
# Backup

backup:
	./zarf/deploy/backup.sh

# ==============================================================================
# Secrets

secrets-edit: ## Edit encrypted secrets (decrypts in editor, re-encrypts on save)
	SOPS_AGE_KEY_FILE=zarf/keys/age.key sops --input-type dotenv --output-type dotenv .secrets.env

secrets-show: ## Print decrypted secrets to stdout
	@SOPS_AGE_KEY_FILE=zarf/keys/age.key sops --decrypt --input-type dotenv --output-type dotenv .secrets.env

secrets-add: ## Usage: make secrets-add KEY=PLANNER_NEW_SECRET VALUE=the-value
	@SOPS_AGE_KEY_FILE=zarf/keys/age.key sops --decrypt --input-type dotenv --output-type dotenv .secrets.env > /tmp/sops-edit.env
	@echo "$(KEY)=$(VALUE)" >> /tmp/sops-edit.env
	@SOPS_AGE_KEY_FILE=zarf/keys/age.key sops --encrypt --input-type dotenv --output-type dotenv /tmp/sops-edit.env > .secrets.env
	@shred -u /tmp/sops-edit.env
	@echo "Added $(KEY) to .secrets.env"

# ==============================================================================
# Dev Tooling

install-hooks:
	cp zarf/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "✓ pre-commit hook installed"

# ==============================================================================
# Code Generation

generate: generate-options generate-kinds
	@echo "✓ TypeScript types generated"

generate-options:
	@mkdir -p api/services/frontend/web/src/types/generated
	go tool tygo generate

generate-kinds:
	go run ./api/tooling/gen-ts-kinds/ api/services/frontend/web/src/types/generated/clarification-kind.ts

# ==============================================================================
# Help

help:
	@echo "Usage:"
	@echo "  make dev-up         - Start DB + migrate + API + Vite (one-shot)"
	@echo "  make dev-up-full    - Start DB + migrate + API + Sidecar + Vite (one-shot)"
	@echo "  make dev-down       - Stop the dev database"
	@echo "  make dev            - Run the API locally"
	@echo "  make admin ARGS=cmd - Run the admin tool"
	@echo "  make migrate        - Run database migrations (local)"
	@echo "  make seed           - Seed the database (local)"
	@echo "  make build          - Build Docker images"
	@echo "  make up             - Start all containers"
	@echo "  make down           - Stop all containers"
	@echo "  make db-up          - Start just the database"
	@echo "  make logs-all       - Tail backend + frontend logs"
	@echo "  make frontend-dev   - Run Vite dev server (proxies /api to :8080)"
	@echo "  make frontend-build - Build frontend"
	@echo "  make frontend-serve - Build and serve frontend via Go SPA server"
	@echo "  make frontend-test  - Run frontend tests"
	@echo "  make frontend-lint  - Lint frontend"
	@echo "  make npm ARGS=...   - Run any npm command in frontend dir"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linter"
	@echo "  make tidy           - Run go mod tidy"
	@echo "  make secrets-edit   - Edit encrypted secrets in $$EDITOR"
	@echo "  make secrets-show   - Print decrypted secrets"
	@echo "  make secrets-add    - Add a secret: KEY=X VALUE=Y"
