PNPM ?= pnpm
GO ?= go
DOCKER ?= docker
COMPOSE ?= docker compose

.PHONY: help
help:
	@printf '%s\n' 'Common targets:'
	@printf '  %-22s %s\n' 'make setup' 'Install frontend dependencies'
	@printf '  %-22s %s\n' 'make test' 'Run backend tests and frontend typecheck'
	@printf '  %-22s %s\n' 'make build' 'Build frontend assets and backend binary'
	@printf '  %-22s %s\n' 'make run' 'Build frontend assets and run gateway'
	@printf '  %-22s %s\n' 'make frontend-dev' 'Run Vite dev server'
	@printf '  %-22s %s\n' 'make docker-build' 'Build the container image'
	@printf '  %-22s %s\n' 'make compose-up' 'Start the deploy compose stack'
	@printf '  %-22s %s\n' 'make compose-logs' 'Follow gateway logs'
	@printf '  %-22s %s\n' 'make compose-down' 'Stop the deploy compose stack'

.PHONY: setup
setup: frontend-install

.PHONY: test
test: backend-test frontend-typecheck

.PHONY: build
build: frontend-build-backend backend-build

.PHONY: run
run: frontend-build-backend
	cd backend && $(GO) run ./cmd/virtual-gamepad

.PHONY: run-no-sudo
run-no-sudo: frontend-build-backend
	cd backend && $(GO) run ./cmd/virtual-gamepad

.PHONY: frontend-install
frontend-install:
	cd frontend && $(PNPM) install --frozen-lockfile

.PHONY: frontend-dev
frontend-dev:
	cd frontend && $(PNPM) dev

.PHONY: frontend-typecheck
frontend-typecheck:
	cd frontend && $(PNPM) run typecheck

.PHONY: frontend-build
frontend-build:
	cd frontend && $(PNPM) build

.PHONY: frontend-build-backend
frontend-build-backend:
	cd frontend && $(PNPM) build:backend

.PHONY: backend-test
backend-test:
	cd backend && $(GO) test ./...

.PHONY: backend-build
backend-build:
	cd backend && $(GO) build -o virtual-gamepad ./cmd/virtual-gamepad

.PHONY: backend-run
backend-run:
	cd backend && $(GO) run ./cmd/virtual-gamepad

.PHONY: backend-run-no-sudo
backend-run-no-sudo:
	cd backend && $(GO) run ./cmd/virtual-gamepad

.PHONY: docker-build
docker-build:
	$(DOCKER) build -t virtual-gamepad:local .

.PHONY: compose-up
compose-up:
	cd deploy && $(COMPOSE) up -d

.PHONY: compose-down
compose-down:
	cd deploy && $(COMPOSE) down

.PHONY: compose-logs
compose-logs:
	cd deploy && $(COMPOSE) logs -f virtual-gamepad

.PHONY: compose-attach
compose-attach:
	$(DOCKER) attach virtual-gamepad
