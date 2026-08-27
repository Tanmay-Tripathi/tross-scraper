MODULE  = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2>/dev/null || echo "0.1.0")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"
GO      := go

SHELL    := /bin/bash
PID_FILE := ./.pid

.PHONY: default
default: help

.PHONY: help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## run the API server
	$(GO) run $(LDFLAGS) ./cmd/server

.PHONY: run-restart
run-restart: ## restart the API server (used by run-live)
	@pkill -P `cat $(PID_FILE)` || true
	@echo "Source changed. Restarting server..."
	@$(GO) run $(LDFLAGS) ./cmd/server & echo $$! > $(PID_FILE)

.PHONY: run-live
run-live: ## run with live reload (requires fswatch)
	@$(GO) run $(LDFLAGS) ./cmd/server & echo $$! > $(PID_FILE)
	@fswatch -x -o --event Created --event Updated --event Renamed -r internal pkg cmd config | xargs -n1 -I {} $(MAKE) run-restart

.PHONY: build
build: ## build the API server binary
	CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o server ./cmd/server

.PHONY: test
test: ## run tests
	$(GO) test ./...

.PHONY: fmt
fmt: ## format code and tidy modules
	$(GO) fmt ./...
	$(GO) mod tidy

.PHONY: vet
vet: ## run static analysis
	$(GO) vet ./...

.PHONY: lint
lint: fmt vet ## format, tidy and vet

.PHONY: clean
clean: ## remove build artifacts
	rm -rf server dist coverage.out $(PID_FILE)

.PHONY: version
version: ## print the build version
	@echo $(VERSION)

.PHONY: up
up: ## start Postgres and Redis for local development
	docker compose up -d postgres redis

.PHONY: down
down: ## stop the local development stack
	docker compose down

.PHONY: stack
stack: ## build and run the full stack (API, Postgres, Redis)
	docker compose up --build

.PHONY: migrate-new
migrate-new: ## create a new Postgres migration pair
	@read -p "Migration name: " name; \
	timestamp=$$(date +%Y%m%d%H%M%S); \
	filename="$${timestamp}_$${name// /_}"; \
	touch "migrations/postgres/$${filename}.up.sql" "migrations/postgres/$${filename}.down.sql"; \
	echo "Created migrations/postgres/$${filename}.{up,down}.sql"

