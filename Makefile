APP_ENV ?= local
GO ?= go
MAIN ?= .
COMPOSE ?= docker compose

.PHONY: help run run-local run-dev run-prod test up down restart ps logs

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
