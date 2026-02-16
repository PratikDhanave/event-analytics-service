# Makefile for event-analytics-service
# Usage examples:
#   make up
#   make test
#   make test-integration
#   make down

SHELL := /bin/bash

# Config (override if needed)
BASE_URL ?= http://localhost:8080
TENANT1_KEY ?= tenant-key-123
TENANT2_KEY ?= tenant-key-456

# Go settings
GO ?= go

# Docker Compose command (works with Docker Compose v2)
DC ?= docker compose

.PHONY: help
help: ## Show available commands
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
	@echo ""
	@echo "Env overrides:"
	@echo "  BASE_URL=$(BASE_URL)"
	@echo "  TENANT1_KEY=$(TENANT1_KEY)"
	@echo "  TENANT2_KEY=$(TENANT2_KEY)"
	@echo ""

.PHONY: tidy
tidy: ## go mod tidy
	$(GO) mod tidy

.PHONY: fmt
fmt: ## gofmt all Go files
	$(GO) fmt ./...

.PHONY: vet
vet: ## go vet
	$(GO) vet ./...

.PHONY: test-unit
test-unit: ## Run unit tests (fast) - excludes /tests integration package
	$(GO) test ./... -v -count=1 | sed '/\/tests/,+1d'

.PHONY: test-integration
test-integration: ## Run integration tests (requires docker compose up)
	BASE_URL=$(BASE_URL) TENANT1_KEY=$(TENANT1_KEY) TENANT2_KEY=$(TENANT2_KEY) \
	$(GO) test ./tests -v -count=1

.PHONY: test
test: ## Run full test flow: bring up stack, wait, run integration tests, then tear down
	$(MAKE) up
	$(MAKE) wait-ready
	$(MAKE) test-integration
	$(MAKE) down

.PHONY: build
build: ## Build the API binary locally
	$(GO) build -o bin/api ./cmd/api

.PHONY: docker-build
docker-build: ## Build docker image via compose
	$(DC) build --no-cache

.PHONY: up
up: ## Start services via docker compose
	$(DC) up --build -d

.PHONY: down
down: ## Stop services via docker compose
	$(DC) down

.PHONY: logs
logs: ## Tail API logs
	$(DC) logs -f api

.PHONY: ps
ps: ## Show running compose services
	$(DC) ps

.PHONY: wait-ready
wait-ready: ## Wait until /ready returns 200
	@echo "Waiting for readiness at $(BASE_URL)/ready ..."
	@for i in {1..60}; do \
		code=$$(curl -s -o /dev/null -w "%{http_code}" "$(BASE_URL)/ready" || true); \
		if [ "$$code" = "200" ]; then \
			echo "Ready ✅"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Not ready ❌"; \
	exit 1

.PHONY: smoke
smoke: ## Quick smoke test: /health and /ready
	@echo "GET /health"
	@curl -s "$(BASE_URL)/health" && echo ""
	@echo "GET /ready"
	@curl -s "$(BASE_URL)/ready" && echo ""

.PHONY: ingest-example
ingest-example: ## Post one sample event (tenant1)
	@curl -s -i -X POST "$(BASE_URL)/events" \
	  -H "X-API-Key: $(TENANT1_KEY)" \
	  -H "Content-Type: application/json" \
	  -H "Idempotency-Key: sample-$$(date +%s)" \
	  -d '{"event_name":"login","timestamp":"2026-02-13T20:00:00Z","properties":{"source":"makefile"}}' | sed -n '1,12p'

.PHONY: metrics-example
metrics-example: ## Query metrics for sample window (tenant1)
	@curl -s "$(BASE_URL)/metrics?event_name=login&from=2026-02-13T00:00:00Z&to=2026-02-14T00:00:00Z" \
	  -H "X-API-Key: $(TENANT1_KEY)" && echo ""
