SHELL := C:/Program Files/Git/usr/bin/bash.exe
.SHELLFLAGS := -c

.PHONY: up down restart logs infra-apply infra-destroy seed test test-unit test-integration test-contract test-frontend lint tidy help

# ─────────────────────────────────────────
# Bootstrap completo
# ─────────────────────────────────────────
up: ## Start everything (LocalStack + infra + services + frontend + observability)
	@echo "==> Copying .env.example to .env if not exists..."
	@cp -n .env.example .env 2>/dev/null || true
	@echo "==> Starting platform..."
	docker compose up -d --build
	@echo "==> Waiting for infra bootstrap..."
	docker wait terraform-init
	@echo "==> Platform is ready!"
	@echo "    Frontend:             http://localhost:5173"
	@echo "    order-service:        http://localhost:3001"
	@echo "    payment-service:      http://localhost:3002"
	@echo "    inventory-service:    http://localhost:3003"
	@echo "    notification-service: http://localhost:3004"
	@echo "    LocalStack:           http://localhost:4566"
	@echo "    Grafana (logs):       http://localhost:3000"
	@echo "    Loki:                 http://localhost:3100"

dev: ## Start with hot reload (air para Go, Vite HMR para frontend)
	@echo "==> Copying .env.example to .env if not exists..."
	@cp -n .env.example .env 2>/dev/null || true
	@echo "==> Starting platform (dev mode com hot reload)..."
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
	@echo "==> Waiting for infra bootstrap..."
	docker wait terraform-init
	@echo "==> Dev mode ativo com hot reload!"
	@echo "    Frontend (Vite HMR):  http://localhost:5173"
	@echo "    order-service:        http://localhost:3001"
	@echo "    Grafana (logs):       http://localhost:3000"

down: ## Stop all containers
	docker compose down

restart: ## Restart all services (keep LocalStack data)
	docker compose restart order-service payment-service inventory-service notification-service frontend

logs: ## Tail logs from all services
	docker compose logs -f order-service payment-service inventory-service notification-service

logs-all: ## Tail all container logs
	docker compose logs -f

infra-apply: ## Apply Terraform (re-provision infra)
	docker compose run --rm terraform-init

infra-destroy: ## Destroy Terraform-managed infra
	docker compose run --rm -e TF_ARGS=destroy terraform-init

seed: ## Seed DynamoDB with sample data
	@echo "==> Seeding data..."
	$(SHELL) scripts/seed.sh

test: ## Run all unit + contract tests (não requer LocalStack)
	$(SHELL) scripts/test-unit.sh

test-unit: ## Run unit tests only (Go + Frontend)
	$(SHELL) scripts/test-unit.sh

test-integration: ## Run integration tests (requer: make up)
	$(SHELL) scripts/test-integration.sh

test-contract: ## Run contract tests only
	$(SHELL) scripts/test-contract.sh

test-frontend: ## Run frontend tests only (Vitest)
	cd frontend && npm test

test-frontend-watch: ## Run frontend tests in watch mode
	cd frontend && npm run test:watch

tidy: ## Run go mod tidy on all services
	@for svc in order-service payment-service inventory-service notification-service; do \
		echo "  --> $$svc"; \
		(cd services/$$svc && go mod tidy); \
	done

lint: ## Lint all services
	$(SHELL) scripts/lint.sh

clean: ## Remove containers, volumes, and LocalStack data
	docker compose down -v
	rm -rf docker/localstack/data

status: ## Show health of all services
	@docker compose ps

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
