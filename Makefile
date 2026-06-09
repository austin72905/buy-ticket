APP_ENV ?= local
GO ?= go
MAIN ?= .
COMPOSE ?= docker compose
MIGRATE ?= migrate
MIGRATIONS_DIR ?= ./migrations
POSTGRES_DSN ?= postgres://postgres:postgres@localhost:5432/buy_ticket?sslmode=disable

.PHONY: help run run-local run-dev run-prod test up down restart ps logs migrate-up migrate-down migrate-drop migrate-force

help:
	@echo "Targets:"
	@echo "  make run APP_ENV=local   # run app with selected APP_ENV"
	@echo "  make run-local           # run app with APP_ENV=local"
	@echo "  make run-dev             # run app with APP_ENV=dev"
	@echo "  make run-prod            # run app with APP_ENV=prod"
	@echo "  make test                # run go test ./..."
	@echo "  make up                  # start docker compose services"
	@echo "  make down                # stop docker compose services"
	@echo "  make restart             # restart docker compose services"
	@echo "  make ps                  # show docker compose services"
	@echo "  make logs                # tail docker compose logs"
	@echo "  make migrate-up          # run all up migrations on local postgres"
	@echo "  make migrate-down        # rollback one migration on local postgres"
	@echo "  make migrate-drop        # drop all database objects on local postgres"
	@echo "  make migrate-force VERSION=1  # force migration version"

run:
	APP_ENV=$(APP_ENV) $(GO) run $(MAIN)

run-local:
	APP_ENV=local $(GO) run $(MAIN)

run-dev:
	APP_ENV=dev $(GO) run $(MAIN)

run-prod:
	APP_ENV=prod $(GO) run $(MAIN)

test:
	$(GO) test ./...

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) down
	$(COMPOSE) up -d

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" down 1

migrate-drop:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" drop -f

migrate-force:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" force $(VERSION)
